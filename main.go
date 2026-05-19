package main

import (
	"nt-simulator/link"
	"nt-simulator/node/host"
	"nt-simulator/node/router"
	"nt-simulator/nteventsched"
)

func main() {
	nes := nteventsched.NewNtEventSched(true, true)
	n1 := host.NewHost(1, "00:1A:2B:3C:4D:5E", "192.168.1.1/24", 1500, nes)
	r1 := router.NewRouter(2, []string{"192.168.1.254/24", "10.1.3.1/24", "10.1.4.1/24"}, nes)
	r2 := router.NewRouter(3, []string{"192.168.2.254/24", "10.2.3.1/24", "10.2.4.1/24"}, nes)
	r3 := router.NewRouter(4, []string{"10.1.3.2/24", "10.2.3.2/24"}, nes)
	r4 := router.NewRouter(5, []string{"10.1.4.2/24", "10.2.4.2/24"}, nes)
	n2 := host.NewHost(6, "00:1A:2B:3C:4D:5F", "192.168.2.1/24", 1500, nes)

	// s1 := nswitch.NewSwitch(nes, 5, "192.168.1.3/24", "00:1A:2B:3C:4D:5E")
	// s2 := nswitch.NewSwitch(nes, 6, "192.170.1.2", "00:1A:2B:3C:3D:5E")
	// s3 := nswitch.NewSwitch(nes, 7, "192.171.1.2", "00:1A:2B:3C:2D:5E")
	// s4 := nswitch.NewSwitch(nes, 8, "192.172.1.2", "00:1A:2B:3C:1D:6E")

	l1 := link.NewLink(n1, r1, 100000, 0.001, 0.0, nes)
	l2 := link.NewLink(r2, n2, 100000, 0.001, 0.0, nes)
	l3 := link.NewLink(r1, r3, 200000, 0.001, 0.0, nes)
	l4 := link.NewLink(r1, r4, 100000, 0.001, 0.0, nes)
	l5 := link.NewLink(r2, r3, 200000, 0.001, 0.0, nes)
	l6 := link.NewLink(r2, r4, 100000, 0.001, 0.0, nes)

	l1.PrintLink()
	l2.PrintLink()
	l3.PrintLink()
	l4.PrintLink()
	l5.PrintLink()
	l6.PrintLink()

	// r1.AddRoute("192.168.1.0/24", "", l1) // destination, nexthop
	// r1.AddRoute("192.168.2.0/24", "10.0.0.2/24", l2)
	// r2.AddRoute("192.168.2.0/24", "", l3)
	// r2.AddRoute("192.168.1.0/24", "10.0.0.1/24", l2)
	// link.NewLink(s3, n2, 100000, 0.001, 0.0, nes)
	// link.NewLink(s1, s2, 100000, 0.001, 0.0, nes)
	// link.NewLink(s1, s3, 100000, 0.001, 0.0, nes)
	// link.NewLink(s1, s4, 100000, 0.001, 0.0, nes)
	// link.NewLink(s2, s3, 100000, 0.001, 0.0, nes)
	// link.NewLink(s2, s4, 100000, 0.001, 0.0, nes)
	// link.NewLink(s3, s4, 100000, 0.001, 0.0, nes)

	// n1.SetTraffic("00:1A:2B:3C:4D:5F", "192.168.2.1/24", 8000, 1.0, 10.0, 40.0, 10000, 1.0)
	// n2.SetTraffic("00:1A:2B:3C:4D:5E", 8000, 40.0, 10.0, 40.0, 85.0, 1.0)
	// linkを繋ぐ前
	// s1.PrintLinkStates()
	// s2.PrintLinkStates()
	nes.RunUntil(50.0)
	// nes.PrintPacketLogs()
	// nes.GenerateSummary()
	// s1.PrintForwadingTable()
	// s2.PrintForwadingTable()
	// linkを繋いだあと
	// s1.PrintLinkStates()
	// s2.PrintLinkStates()
	nes.Visualize()
}
