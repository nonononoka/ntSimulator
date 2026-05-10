package main

import (
	"fmt"
	"nt-simulator/link"
	"nt-simulator/node/host"
	"nt-simulator/node/nswitch"
	"nt-simulator/nteventsched"
)

func main() {
	nes := nteventsched.NewNtEventSched(true, true)
	n1, err1 := host.NewHost(1, "00:1A:2B:3C:4D:5E", "192.168.1.1", 1500, nes)
	if err1 != nil {
		fmt.Println(err1)
		return
	}
	n2, err2 := host.NewHost(2, "00:1A:2B:3C:4D:5F", "192.168.1.2", 1500, nes)
	if err2 != nil {
		fmt.Println(err2)
		return
	}
	// n3, err3 := network.NewNode(3, "00:1A:2B:3C:4D:5C", nes)
	// if err3 != nil {
	// 	fmt.Println(err2)
	// 	return
	// }
	// n4, err4 := network.NewNode(4, "00:1A:2B:3C:4D:5D", nes)
	// if err4 != nil {
	// 	fmt.Println(err4)
	// 	return
	// }
	s1 := nswitch.NewSwitch(nes, 5)
	s2 := nswitch.NewSwitch(nes, 6)
	s3 := nswitch.NewSwitch(nes, 7)
	s4 := nswitch.NewSwitch(nes, 8)

	link.NewLink(s1, n1, 100000, 0.001, 0.0, nes)
	link.NewLink(s3, n2, 100000, 0.001, 0.0, nes)
	link.NewLink(s1, s2, 100000, 0.001, 0.0, nes)
	link.NewLink(s1, s3, 100000, 0.001, 0.0, nes)
	link.NewLink(s1, s4, 100000, 0.001, 0.0, nes)
	link.NewLink(s2, s3, 100000, 0.001, 0.0, nes)
	link.NewLink(s2, s4, 100000, 0.001, 0.0, nes)
	link.NewLink(s3, s4, 100000, 0.001, 0.0, nes)

	n1.SetTraffic("00:1A:2B:3C:4D:5F", "192.168.1.2", 8000, 1.0, 10.0, 40.0, 10000, 1.0)
	// n2.SetTraffic("00:1A:2B:3C:4D:5E", 8000, 40.0, 10.0, 40.0, 85.0, 1.0)
	// linkを繋ぐ前
	s1.PrintLinkStates()
	s2.PrintLinkStates()
	nes.Run()
	// nes.PrintPacketLogs()
	nes.GenerateSummary()
	s1.PrintForwadingTable()
	s2.PrintForwadingTable()
	// linkを繋いだあと
	s1.PrintLinkStates()
	s2.PrintLinkStates()
	nes.Visualize()
}
