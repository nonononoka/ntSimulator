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
	n3, err3 := network.NewNode(3, "00:1A:2B:3C:4D:5C", nes)
	if err3 != nil {
		fmt.Println(err2)
		return
	}
	n4, err4 := network.NewNode(4, "00:1A:2B:3C:4D:5D", nes)
	if err4 != nil {
		fmt.Println(err4)
		return
	}
	s1 := network.NewSwitch(nes, 5)
	s1.PrintForwadingTable()
	s2 := network.NewSwitch(nes, 6)
	// s1を介して，n1~n4が繋がっている
	l1 := network.NewLink(s1, n1, 100000, 0.001, 0.0, nes)
	l2 := network.NewLink(s1, n2, 100000, 0.001, 0.0, nes)
	l3 := network.NewLink(s2, n3, 100000, 0.001, 0.0, nes)
	l4 := network.NewLink(s2, n4, 100000, 0.001, 0.0, nes)

	network.NewLink(s1, s2, 100000, 0.001, 0.0, nes)

	n1.PrintNode()
	n2.PrintNode()
	n3.PrintNode()
	n4.PrintNode()
	l1.PrintLink()
	l2.PrintLink()
	l3.PrintLink()
	l4.PrintLink()

	n1.SetTraffic("00:1A:2B:3C:4D:5D", 8000, 1.0, 10.0, 40.0, 85.0, 1.0)
	// n2.SetTraffic("00:1A:2B:3C:4D:5E", 8000, 40.0, 10.0, 40.0, 85.0, 1.0)
	nes.Visualize()
	// linkを繋ぐ前
	s1.PrintLinkStates()
	s2.PrintLinkStates()
	nes.Run()
	nes.PrintPacketLogs()
	nes.GenerateSummary()
	s1.PrintForwadingTable()
	s2.PrintForwadingTable()
	// linkを繋いだあと
	s1.PrintLinkStates()
	s2.PrintLinkStates()
}
