package main

import (
	"nt-simulator/network"
	"nt-simulator/nteventsched"
)

func main() {
	nes := nteventsched.NewNtEventSched(true, true)
	n1 := network.NewNode(1, "00:01", nes)
	n2 := network.NewNode(2, "00:02", nes)
	l1 := network.NewLink(n1, n2, 10000, 0.001, 0.6, nes)
	n1.PrintNode()
	n2.PrintNode()
	l1.PrintLink()

	headerSize := 40
	payloadSize := 85
	n1.SetTraffic("00:02", 8000, 1.0, 10.0, float64(headerSize), float64(payloadSize), 1.0)
	nes.Visualize()
	nes.Run()
	nes.PrintPacketLogs()
	nes.GenerateSummary()
}
