package main

import (
	"fmt"
	"nt-simulator/network"
	"nt-simulator/nteventsched"
)

func main() {
	nes := nteventsched.NewNtEventSched(true, true)
	n1, err1 := network.NewNode(1, "00:1A:2B:3C:4D:5E", nes)
	if err1 != nil {
		fmt.Println(err1)
		return
	}
	n2, err2 := network.NewNode(2, "00:1A:2B:3C:4D:5F", nes)
	if err2 != nil {
		fmt.Println(err2)
		return
	}
	s1 := network.NewSwitch(nes, 3)
	l1 := network.NewLink(s1, n1, 100000, 0.001, 0.0, nes)
	l2 := network.NewLink(s1, n2, 100000, 0.001, 0.0, nes)

	s1.UpdateForwardingTable(n1.Address(), l1) // n1にいくならl1のリンク
	s1.UpdateForwardingTable(n2.Address(), l2) // n2にいくならl2のリンク
	n1.PrintNode()
	n2.PrintNode()
	l1.PrintLink()
	l2.PrintLink()

	n1.SetTraffic("00:1A:2B:3C:4D:5F", 8000, 1.0, 10.0, 40.0, 85.0, 1.0)
	nes.Visualize()
	nes.Run()
	nes.PrintPacketLogs()
	nes.GenerateSummary()
}
