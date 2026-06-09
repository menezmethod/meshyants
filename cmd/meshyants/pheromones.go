package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

// pheromonesCmd tails mesh.phero.<trust-domain> and prints each PheromoneRecord as one JSON line.
func pheromonesCmd(args []string) {
	fs := flag.NewFlagSet("pheromones", flag.ExitOnError)
	td := fs.String("trust-domain", envOr("MESHYANTS_TRUST_DOMAIN", "default"), "trust domain (subject mesh.phero.<td>)")
	fs.SetOutput(os.Stderr)
	fs.Parse(args)

	natsURL := envOr("NATS_URL", "nats://localhost:4222")
	nc, err := nats.Connect(natsURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pheromones: connect %s: %v\n", natsURL, err)
		os.Exit(1)
	}
	defer nc.Drain()

	subj := fmt.Sprintf("mesh.phero.%s", *td)
	_, err = nc.Subscribe(subj, func(msg *nats.Msg) {
		var rec meshyantsv1.PheromoneRecord
		if err := proto.Unmarshal(msg.Data, &rec); err != nil {
			fmt.Fprintf(os.Stderr, "pheromones: skip non-protobuf message on %s: %v\n", subj, err)
			return
		}
		line := map[string]any{
			"record_id":      rec.RecordId,
			"kind":           rec.Kind.String(),
			"subject":        rec.Subject,
			"issuer":         rec.Issuer,
			"trust_domain":   rec.TrustDomain,
			"causal_parents": rec.CausalParents,
		}
		if rec.IssuedAt != nil {
			line["issued_at"] = rec.IssuedAt.AsTime().UTC().Format(time.RFC3339Nano)
		}
		b, err := json.Marshal(line)
		if err != nil {
			return
		}
		fmt.Println(string(b))
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "pheromones: subscribe %s: %v\n", subj, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "pheromones: listening on %s (ctrl-c to stop)\n", subj)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
