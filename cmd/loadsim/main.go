// Command loadsim runs N simulated devices against a real NATS+JetStream broker.
//
// Two modes:
//
//   - Default (in-process): each "device" is a goroutine running agent.Worker.
//     Fast to iterate on; does not exercise the real binary, argv parsing, or
//     process isolation — only the library path.
//
//   - -publish-only: signs and publishes tasks only. You run N real
//     `meshyants worker` processes (e.g. one per simulated LicheeRV Nano) with
//     matching --subject, --issuer=oracle, and the same --key-file as -key-file
//     here. That matches field layout: real OS process, real executor flags.
//
// Example (real workers + loadsim as oracle traffic):
//
//	openssl rand -out oracle.key 32
//	for i in $(seq 0 9); do
//	  NATS_URL=nats://127.0.0.1:4222 meshyants worker --trust-domain=load \
//	    --subject=mesh.task.load.dev$i --issuer=oracle --key-file=oracle.key \
//	    --executor=native --effect-db=/tmp/mesh-fleet-$i.db &
//	done
//	go run ./cmd/loadsim -nats=nats://127.0.0.1:4222 -publish-only -key-file=oracle.key -devices=10 -tasks=100
//
// Example (all-in-one smoke, in-process):
//
//	docker run --rm -p 4222:4222 nats:2.10-alpine -js
//	go run ./cmd/loadsim -nats nats://127.0.0.1:4222 -devices 50 -tasks 500 -quiet
package main

import (
	"context"
	"crypto/ed25519"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/agent"
	"github.com/meshyants/meshyants/v1/internal/execwasm"
	"github.com/meshyants/meshyants/v1/internal/fabric"
	"github.com/meshyants/meshyants/v1/internal/ledger/effect"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	natsURL := flag.String("nats", os.Getenv("NATS_URL"), "NATS URL (required), e.g. nats://127.0.0.1:4222")
	devices := flag.Int("devices", 10, "number of devices (subject suffix 0..N-1); in-process workers or external fleet size")
	tasks := flag.Int("tasks", 100, "total signed tasks to publish (round-robin across devices)")
	trust := flag.String("trust", "load", "trust domain (subjects mesh.task.<trust>.devN, mesh.phero.<trust>)")
	wasm := flag.Bool("wasm", false, "use Wasm executor (slower); default is native executor")
	quiet := flag.Bool("quiet", false, "suppress worker and publisher logs")
	timeout := flag.Duration("timeout", 5*time.Minute, "max wait for all SAFE pheromones")
	publishOnly := flag.Bool("publish-only", false, "only publish tasks; run real meshyants worker processes separately (requires -key-file)")
	keyFile := flag.String("key-file", "", "Ed25519 private key for signing tasks (32 raw bytes or base64); required with -publish-only; optional in default mode (ephemeral if empty)")
	flag.Parse()

	if *natsURL == "" {
		fmt.Fprintln(os.Stderr, "loadsim: -nats or NATS_URL is required")
		flag.Usage()
		os.Exit(2)
	}
	if *devices < 1 || *tasks < 1 {
		fmt.Fprintln(os.Stderr, "loadsim: -devices and -tasks must be >= 1")
		os.Exit(2)
	}
	if *publishOnly && *keyFile == "" {
		fmt.Fprintln(os.Stderr, "loadsim: -publish-only requires -key-file (same file passed to each meshyants worker)")
		os.Exit(2)
	}

	if *quiet {
		log.SetOutput(io.Discard)
	}

	ctx := context.Background()
	runID := uuid.New().String()[:8]

	var oraclePub ed25519.PublicKey
	var oraclePriv ed25519.PrivateKey
	var err error
	if *keyFile != "" {
		oraclePriv, err = signing.LoadPrivateKeyFromFile(*keyFile)
		if err != nil {
			log.Fatal(err)
		}
		oraclePub = oraclePriv.Public().(ed25519.PublicKey)
	} else {
		oraclePub, oraclePriv, err = signing.GenerateKeyPair()
		if err != nil {
			log.Fatal(err)
		}
	}

	var tmpRoot string
	if !*publishOnly {
		tmpRoot, err = os.MkdirTemp("", "meshyants-loadsim-*")
		if err != nil {
			log.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(tmpRoot) }()
	}

	pubTransport, err := fabric.ConnectJetStream(ctx, *natsURL, fabric.DefaultJetStreamConfig())
	if err != nil {
		log.Fatalf("connect publisher NATS: %v", err)
	}
	defer pubTransport.Close()

	witnessNC, err := nats.Connect(*natsURL)
	if err != nil {
		log.Fatalf("connect witness NATS: %v", err)
	}
	defer func() { _ = witnessNC.Drain() }()
	jsWitness, err := witnessNC.JetStream()
	if err != nil {
		log.Fatalf("jetstream witness: %v", err)
	}

	pheroSubject := fmt.Sprintf("mesh.phero.%s", *trust)
	witnessDurable := fmt.Sprintf("loadsim-witness-%s", runID)
	pheroSub, err := jsWitness.PullSubscribe(pheroSubject, witnessDurable, nats.BindStream("MESHYANTS"), nats.DeliverAll())
	if err != nil {
		log.Fatalf("pheromone witness subscribe: %v", err)
	}
	defer func() { _ = pheroSub.Unsubscribe() }()

	workerCtx, stopWorkers := context.WithCancel(ctx)
	var wg sync.WaitGroup

	if !*publishOnly {
		for i := 0; i < *devices; i++ {
			i := i
			subj := fmt.Sprintf("mesh.task.%s.dev%d", *trust, i)
			dbPath := filepath.Join(tmpRoot, fmt.Sprintf("effect-%d.db", i))

			tr, err := fabric.ConnectJetStream(ctx, *natsURL, fabric.DefaultJetStreamConfig())
			if err != nil {
				log.Fatalf("device %d connect: %v", i, err)
			}

			ledger, err := effect.Open(dbPath)
			if err != nil {
				log.Fatalf("device %d ledger: %v", i, err)
			}

			var exec agent.Executor = agent.NativeExecutor{}
			if *wasm {
				exec = agent.WasmExecutor{Quotas: execwasm.DefaultQuotas()}
			}

			w := &agent.Worker{
				Transport:    tr,
				EffectLedger: ledger,
				Executor:     exec,
				PublicKey:    oraclePub,
				PrivateKey:   oraclePriv,
				Issuer:       "oracle",
				TrustDomain:  *trust,
				Subject:      subj,
				Durable:      fmt.Sprintf("load-%s-%d", runID, i),
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { _ = ledger.Close() }()
				defer tr.Close()
				if err := w.Run(workerCtx); err != nil && err != context.Canceled {
					log.Printf("worker dev%d exit: %v", i, err)
				}
			}()
		}
	}

	time.Sleep(300 * time.Millisecond)

	var safeCount atomic.Int64
	done := make(chan struct{})
	go func() {
		defer close(done)
		deadline := time.Now().Add(*timeout)
		for safeCount.Load() < int64(*tasks) && time.Now().Before(deadline) {
			want := int(*tasks) - int(safeCount.Load())
			if want > 256 {
				want = 256
			}
			if want < 1 {
				want = 1
			}
			batch, err := pheroSub.Fetch(want, nats.MaxWait(time.Second))
			if err != nil && err != nats.ErrTimeout {
				return
			}
			for _, m := range batch {
				var rec meshyantsv1.PheromoneRecord
				if err := proto.Unmarshal(m.Data, &rec); err != nil {
					continue
				}
				if rec.Kind == meshyantsv1.PheromoneKind_PHEROMONE_KIND_SAFE {
					safeCount.Add(1)
				}
			}
		}
	}()

	t0 := time.Now()
	for k := 0; k < *tasks; k++ {
		dev := k % *devices
		subj := fmt.Sprintf("mesh.task.%s.dev%d", *trust, dev)
		taskID := fmt.Sprintf("task-%s-%d", runID, k)
		idem := fmt.Sprintf("idem-%s-%d", runID, k)

		task := &meshyantsv1.TaskAtom{
			SchemaVersion:    1,
			TaskId:           taskID,
			IdempotencyKey:   idem,
			Issuer:           "oracle",
			TrustDomain:      *trust,
			CausalParents:    []string{"genesis"},
			RequirementsJson: `{"transport":"fast-local"}`,
			Payload:          []byte(`{"load":true}`),
			ExpiresAt:        timestamppb.New(time.Now().Add(time.Hour)),
		}
		if *wasm {
			task.Payload = execwasm.ValidWasm()
		}
		if err := signing.Sign(task, oraclePriv); err != nil {
			log.Fatalf("sign task %d: %v", k, err)
		}
		payload, err := proto.Marshal(task)
		if err != nil {
			log.Fatalf("marshal task %d: %v", k, err)
		}
		if err := pubTransport.Publish(ctx, subj, payload, fabric.PublishOpts{MsgID: fabric.StableMsgID(idem)}); err != nil {
			log.Fatalf("publish task %d: %v", k, err)
		}
	}

	<-done
	elapsed := time.Since(t0)
	stopWorkers()
	wg.Wait()

	got := int(safeCount.Load())
	mode := "in-process"
	if *publishOnly {
		mode = "publish-only"
	}
	fmt.Printf("loadsim: mode=%s devices=%d tasks=%d wasm=%v elapsed=%s rate=%.1f task/s SAFE=%d/%d\n",
		mode, *devices, *tasks, *wasm, elapsed.Round(time.Millisecond), float64(*tasks)/elapsed.Seconds(), got, *tasks)

	if got < *tasks {
		fmt.Fprintf(os.Stderr, "loadsim: incomplete — expected %d SAFE pheromones, got %d (timeout %s)\n", *tasks, got, *timeout)
		os.Exit(1)
	}
}
