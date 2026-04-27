package main

import (
	"nt-simulator/node"
	"nt-simulator/nteventsched"
)

func main() {
	nes := nteventsched.NewNtEventSched(true, true)
	n1 := node.NewNode(1, "00:01", nes)
	n2 := node.NewNode(2, "00:02", nes)
	l1 := node.NewLink(n1, n2, 10000, 0.001, 0.0, nes)
	n1.PrintNode()
	n2.PrintNode()
	l1.PrintLink()

	headerSize := 40
	payloadSize := 85
	n1.SetTraffic("00:02", 1000, 1.0, 10.0, float64(headerSize), float64(payloadSize), 1.0)
	nes.Visualize()
	nes.Run()
}
