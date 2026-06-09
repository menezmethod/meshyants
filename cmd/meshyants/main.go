// Command meshyants is the MeshyAnts agent CLI (docs/v1/02-installation-onboarding-and-governance.md).
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/agent"
	"github.com/meshyants/meshyants/v1/internal/execwasm"
	"github.com/meshyants/meshyants/v1/internal/fabric"
	"github.com/meshyants/meshyants/v1/internal/ledger/effect"
	"github.com/meshyants/meshyants/v1/internal/oracle"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"github.com/meshyants/meshyants/v1/internal/runtime/tier"
	"github.com/google/uuid"
)

const version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "version":
		fmt.Println(version)
	case "doctor":
		doctor()
	case "oracle":
		oracleCmd(os.Args[2:])
	case "worker":
		workerCmd(os.Args[2:])
	case "agent":
		agentCmd(os.Args[2:])
	case "pheromones":
		pheromonesCmd(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `MeshyAnts CLI — autonomous edge-agent swarm

usage: meshyants <command> [args]

commands:
  version                      print version
  doctor                       check runtime tier
  oracle [flags] "<goal>"      send a goal (-key-file, -verbose; env: MINIMAX_API_KEY / OPENAI_API_KEY)
  worker --trust-domain=<td>   run a worker (-verbose; see --subject, --issuer, --key-file, --effect-db)
  agent --trust-domain=<td>    run oracle + worker (-verbose, --goal)
  pheromones --trust-domain=<td>  tail mesh.phero.<td> as JSON lines (worker/oracle outcomes)

examples:
  meshyants oracle "deploy my app to staging"
  # separate oracle + worker: same Ed25519 seed file; worker must use --issuer=oracle
  head -c 32 /dev/urandom > /tmp/mesh.key
  meshyants oracle -key-file=/tmp/mesh.key "deploy to staging"
  meshyants worker --trust-domain=default --issuer=oracle --key-file=/tmp/mesh.key --executor=native
  meshyants worker --trust-domain=dev
  meshyants agent --trust-domain=dev
`)
}

func doctor() {
	ctx := context.Background()
	tr := tier.Detect(ctx, tier.DefaultOptions())
	out := map[string]any{
		"version":      version,
		"runtime_tier": tr.String(),
		"ok":           true,
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}

// --- oracle command ---

func oracleCmd(args []string) {
	fs := flag.NewFlagSet("oracle", flag.ExitOnError)
	keyFile := fs.String("key-file", "", "Ed25519 private key file (32-byte seed); default ephemeral key (not verifiable by a separate worker)")
	verbose := fs.Bool("verbose", false, "log canonicalization system prompt, user goal, and TaskIntentHeader JSON")
	fs.SetOutput(os.Stderr)
	fs.Parse(args)
	rem := fs.Args()
	if len(rem) != 1 {
		fmt.Fprintln(os.Stderr, "usage: meshyants oracle [-key-file=<path>] \"<goal>\"")
		os.Exit(2)
	}
	goal := rem[0]
	ctx := context.Background()

	// Connect to NATS
	natsURL := envOr("NATS_URL", "nats://localhost:4222")
	tr, err := fabric.ConnectJetStream(ctx, natsURL, fabric.DefaultJetStreamConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "oracle: connect NATS at %s: %v\n", natsURL, err)
		os.Exit(1)
	}
	defer tr.Close()

	var pub ed25519.PublicKey
	var priv ed25519.PrivateKey
	if *keyFile != "" {
		priv, err = signing.LoadPrivateKeyFromFile(*keyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "oracle: load key: %v\n", err)
			os.Exit(1)
		}
		pub = priv.Public().(ed25519.PublicKey)
	} else {
		pub, priv, err = signing.GenerateKeyPair()
		if err != nil {
			fmt.Fprintf(os.Stderr, "oracle: generate key pair: %v\n", err)
			os.Exit(1)
		}
	}

	td := envOr("MESHYANTS_TRUST_DOMAIN", "default")

	adapter, backend := oracleAdapterFromEnv()
	switch backend {
	case "minimax":
		fmt.Fprintf(os.Stderr, "oracle: using MiniMax adapter (model=%s)\n", envOr("MINIMAX_MODEL", "MiniMax-M2.7"))
	case "openai":
		fmt.Fprintf(os.Stderr, "oracle: using OpenAI adapter (model=%s)\n", envOr("OPENAI_MODEL", "gpt-4o"))
	default:
		fmt.Fprintf(os.Stderr, "oracle: no MINIMAX_API_KEY or OPENAI_API_KEY, using mock adapter (goal passed through as-is)\n")
	}

	svc := oracle.NewService(adapter, tr, priv, pub, "oracle", td)
	svc.Verbose = *verbose

	fmt.Fprintf(os.Stderr, "oracle: sending goal → %q\n", goal)
	task, phero, err := svc.HandleGoal(ctx, goal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oracle: HandleGoal: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "oracle: published task %q (issuer=%s, trust_domain=%s)\n", task.TaskId, task.Issuer, task.TrustDomain)
	fmt.Fprintf(os.Stderr, "oracle: published TODO pheromone %q\n", phero.RecordId)
	fmt.Printf("ok — task %s enqueued, workers will pick it up on mesh.task.%s\n", task.TaskId, td)
}

// --- worker command ---

func workerCmd(args []string) {
	fs := flag.NewFlagSet("worker", flag.ContinueOnError)
	td := fs.String("trust-domain", envOr("MESHYANTS_TRUST_DOMAIN", "default"), "trust domain")
	executorType := fs.String("executor", "", "executor type: wasm, native (default: auto-detect runtime)")
	subjectFlag := fs.String("subject", "", "JetStream task subject (default mesh.task.<trust-domain>)")
	issuerFlag := fs.String("issuer", "worker", "TaskAtom issuer to verify and pheromone issuer label")
	keyFile := fs.String("key-file", "", "Ed25519 private key (32 raw bytes or base64); default ephemeral key")
	effectPath := fs.String("effect-db", "", "effect ledger Bolt DB path; default /tmp/meshyants_effect_<uuid>.db")
	verbose := fs.Bool("verbose", false, "log JSON line per task outcome (executor detail, intent payload)")
	fs.Parse(args)

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

	natsURL := envOr("NATS_URL", "nats://localhost:4222")
	tr, err := fabric.ConnectJetStream(ctx, natsURL, fabric.DefaultJetStreamConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "worker: connect NATS at %s: %v\n", natsURL, err)
		os.Exit(1)
	}
	defer tr.Close()

	var pub ed25519.PublicKey
	var priv ed25519.PrivateKey
	if *keyFile != "" {
		priv, err = signing.LoadPrivateKeyFromFile(*keyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "worker: load key: %v\n", err)
			os.Exit(1)
		}
		pub = priv.Public().(ed25519.PublicKey)
	} else {
		pub, priv, err = signing.GenerateKeyPair()
		if err != nil {
			fmt.Fprintf(os.Stderr, "worker: generate key pair: %v\n", err)
			os.Exit(1)
		}
	}

	effectDB := *effectPath
	if effectDB == "" {
		effectDB = fmt.Sprintf("/tmp/meshyants_effect_%s.db", uuid.New().String())
	}
	ledger, err := effect.Open(effectDB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "worker: open effect ledger: %v\n", err)
		os.Exit(1)
	}
	defer ledger.Close()

	var executor agent.Executor
	switch *executorType {
	case "wasm":
		executor = agent.WasmExecutor{Quotas: execwasm.DefaultQuotas()}
	case "native":
		executor = agent.NativeExecutor{}
	case "":
		rt := tier.Detect(ctx, tier.DefaultOptions())
		switch rt {
		case meshyantsv1.RuntimeTier_RUNTIME_TIER_EXECUTOR_WASM:
			executor = agent.WasmExecutor{Quotas: execwasm.DefaultQuotas()}
		default:
			executor = agent.NativeExecutor{}
		}
	}

	subject := *subjectFlag
	if subject == "" {
		subject = fmt.Sprintf("mesh.task.%s", *td)
	}
	w := &agent.Worker{
		Transport:    tr,
		EffectLedger: ledger,
		Executor:     executor,
		PublicKey:    pub,
		PrivateKey:   priv,
		Issuer:       *issuerFlag,
		TrustDomain:  *td,
		Subject:      subject,
		Durable:      "worker-" + uuid.New().String()[:8],
		Verbose:      *verbose,
	}

	fmt.Fprintf(os.Stderr, "worker: starting (trust_domain=%s, subject=%s, durable=%s, issuer=%s)\n", *td, subject, w.Durable, w.Issuer)
	if err := w.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "worker: %v\n", err)
		os.Exit(1)
	}
}

// --- agent command (oracle + worker in one process) ---

func agentCmd(args []string) {
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	td := fs.String("trust-domain", envOr("MESHYANTS_TRUST_DOMAIN", "default"), "trust domain")
	goal := fs.String("goal", "", "initial goal to send (optional)")
	verbose := fs.Bool("verbose", false, "oracle: log prompt + TaskIntentHeader; worker: log JSON result per task")
	fs.Parse(args)

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

	natsURL := envOr("NATS_URL", "nats://localhost:4222")
	tr, err := fabric.ConnectJetStream(ctx, natsURL, fabric.DefaultJetStreamConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent: connect NATS at %s: %v\n", natsURL, err)
		os.Exit(1)
	}
	defer tr.Close()

	pub, priv, err := signing.GenerateKeyPair()
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent: generate key pair: %v\n", err)
		os.Exit(1)
	}

	ledger, err := effect.Open(fmt.Sprintf("/tmp/meshyants_effect_%s.db", uuid.New().String()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent: open effect ledger: %v\n", err)
		os.Exit(1)
	}
	defer ledger.Close()

	// Oracle service (MiniMax takes precedence over OpenAI when both keys are set)
	adapter, backend := oracleAdapterFromEnv()
	switch backend {
	case "minimax":
		fmt.Fprintf(os.Stderr, "agent: using MiniMax adapter (model=%s)\n", envOr("MINIMAX_MODEL", "MiniMax-M2.7"))
	case "openai":
		fmt.Fprintf(os.Stderr, "agent: using OpenAI adapter (model=%s)\n", envOr("OPENAI_MODEL", "gpt-4o"))
	default:
		fmt.Fprintf(os.Stderr, "agent: no MINIMAX_API_KEY or OPENAI_API_KEY, using mock adapter (native executor)\n")
	}
	svc := oracle.NewService(adapter, tr, priv, pub, "oracle", *td)
	svc.Verbose = *verbose

	// Worker: HandleGoal always publishes JSON intent in TaskAtom.Payload, not a .wasm module.
	// Tier-based WasmExecutor would treat that blob as Wasm and fail (e.g. "invalid magic number").
	executor := agent.NativeExecutor{}

	subject := fmt.Sprintf("mesh.task.%s", *td)
	w := &agent.Worker{
		Transport:    tr,
		EffectLedger: ledger,
		Executor:     executor,
		PublicKey:   pub,
		PrivateKey:  priv,
		Issuer:      "oracle",
		TrustDomain: *td,
		Subject:    subject,
		Durable:    "agent-" + uuid.New().String()[:8],
		Verbose:    *verbose,
	}

	// If --goal provided, send it AFTER worker starts (so subscription is active)
	// Start worker goroutine first so it begins subscribing
	fmt.Fprintf(os.Stderr, "agent: starting worker on subject=%q (ctrl-c to stop)\n", subject)
	workerErrCh := make(chan error, 1)
	go func() { workerErrCh <- w.Run(ctx) }()
	time.Sleep(2 * time.Second) // let subscription establish

	if *goal != "" {
		fmt.Fprintf(os.Stderr, "agent: oracle sending goal → %q\n", *goal)
		task, phero, err := svc.HandleGoal(ctx, *goal)
		if err != nil {
			fmt.Fprintf(os.Stderr, "agent: HandleGoal: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "agent: task=%s, pheromone=%s\n", task.TaskId, phero.RecordId)
	}

	// Block until worker returns
	err = <-workerErrCh
	if err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "agent: worker error: %v\n", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// oracleAdapterFromEnv picks MiniMax (Token Plan / OpenAI-compatible endpoint) if MINIMAX_API_KEY is set,
// else OpenAI if OPENAI_API_KEY is set, else mock.
func oracleAdapterFromEnv() (oracle.Adapter, string) {
	if k := os.Getenv("MINIMAX_API_KEY"); k != "" {
		return oracle.NewMiniMaxAdapter(oracle.Config{
			APIKey:         k,
			Model:          envOr("MINIMAX_MODEL", "MiniMax-M2.7"),
			RequestTimeout: 60 * time.Second,
			MaxTokens:      1024,
		}), "minimax"
	}
	if k := os.Getenv("OPENAI_API_KEY"); k != "" {
		return oracle.NewOpenAIAdapter(oracle.Config{
			APIKey:         k,
			Model:          envOr("OPENAI_MODEL", "gpt-4o"),
			RequestTimeout: 30 * time.Second,
			MaxTokens:      512,
		}), "openai"
	}
	return &oracle.MockAdapter{}, "mock"
}
