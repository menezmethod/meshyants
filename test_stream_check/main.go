package main

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

func main() {
	nc, _ := nats.Connect("nats://localhost:4222")
	js, _ := nc.JetStream()

	info, err := js.StreamInfo("MESHYANTS")
	if err != nil {
		fmt.Println("Stream error:", err)
	} else {
		fmt.Printf("Stream msgs: %d, bytes: %d\n", info.State.Msgs, info.State.Bytes)
	}

	fmt.Printf("Consumers: %v\n", js.ConsumerNames("MESHYANTS"))
	nc.Drain()
}
