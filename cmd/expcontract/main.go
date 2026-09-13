// Command expcontract validates a MESH-101 experiment run spec.
package main

import (
	"fmt"
	"os"

	"github.com/meshyants/meshyants/v1/internal/expcontract"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "validate" {
		fmt.Fprintln(os.Stderr, "usage: expcontract validate <run-spec.json>")
		os.Exit(2)
	}
	spec, err := expcontract.Load(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "expcontract: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("ok\t%s\tphase=%s\tclass=%s\tarms=%d\ttasks=%d\tpool=%d\n",
		spec.RunID, spec.Phase, spec.Workload.Class, len(spec.Systems), len(spec.Tasks.Tasks), len(spec.Pool.Instances))
}
