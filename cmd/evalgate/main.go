// Command evalgate validates evaluator reports, decides whether work may
// advance, and writes spawned task drafts from negative findings.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/meshyants/meshyants/v1/internal/evalgate"
)

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	path := os.Args[2]
	report, err := evalgate.LoadReport(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "evalgate: load: %v\n", err)
		os.Exit(1)
	}

	switch cmd {
	case "validate":
		if err := evalgate.Validate(report); err != nil {
			fmt.Fprintf(os.Stderr, "evalgate: %v\n", err)
			os.Exit(2)
		}
		fmt.Println("ok")
	case "decide":
		d, err := evalgate.Decide(report)
		if err != nil {
			fmt.Fprintf(os.Stderr, "evalgate: %v\n", err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(d); err != nil {
			fmt.Fprintf(os.Stderr, "evalgate: encode: %v\n", err)
			os.Exit(1)
		}
		if !d.Advance {
			os.Exit(2)
		}
	case "spawn":
		if err := evalgate.Validate(report); err != nil {
			fmt.Fprintf(os.Stderr, "evalgate: %v\n", err)
			os.Exit(1)
		}
		drafts := evalgate.Spawn(report)
		out := "-"
		if len(os.Args) >= 5 && os.Args[3] == "-o" {
			out = os.Args[4]
		}
		if out == "-" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(drafts)
			return
		}
		if err := evalgate.WriteSpawned(out, drafts); err != nil {
			fmt.Fprintf(os.Stderr, "evalgate: write: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %d drafts to %s\n", len(drafts), out)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage:\n  evalgate validate <report.json>\n  evalgate decide <report.json>\n  evalgate spawn <report.json> [-o drafts.json]\n")
}
