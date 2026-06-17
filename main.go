package main

import (
	"nt-simulator/link"
	"nt-simulator/node/dhcpserver"
	"nt-simulator/node/dnsserver"
	"nt-simulator/node/host"
	"nt-simulator/node/nswitch"
	"nt-simulator/node/router"
	"nt-simulator/nteventsched"
)

func main() {
	nes := nteventsched.NewNtEventSched(true, true)
	n1 := host.NewHost(1, "192.168.1.0/24", 1500, nes)
	n2 := host.NewHost(2, "192.168.2.1/24", 1500, nes)
	// n3 := host.NewHost(3, "192.168.1.3/24", 1500, nes)
	// n4 := host.NewHost(4, "192.168.1.4/24", 1500, nes)
	r1 := router.NewRouter(3, []string{"192.168.1.254/24", "192.168.2.254/24"}, nes)
	dns1 := dnsserver.NewDNSServer(nes, 4, "192.168.1.200/24")
	// r2 := router.NewRouter(4, []string{"192.168.2.254/24", "10.1.1.2/24"}, nes)
	// r3 := router.NewRouter(4, []string{"10.2.3.3/24", "10.3.4.3/24"}, nes)
	// r4 := router.NewRouter(5, []string{"192.168.2.254/24", "10.3.4.4/24"}, nes)
	s1 := nswitch.NewSwitch(nes, 5, "192.168.1.240/24")
	dhcp1 := dhcpserver.NewDHCPServer(nes, 6, "192.168.1.250/24", "192.168.1.200/24", "192.168.1.0/24")
	dhcp1.ReserveIP(s1.GetIPAddresses()[0].String())
	dhcp1.ReserveIP(dns1.GetIPAddresses()[0].String())
	for _, ip := range r1.GetIPAddresses() {
		dhcp1.ReserveIP(ip.String())
	}
	dhcp1.ReserveIP("192.168.1.250/24")
	l1 := link.NewLink(n1, s1, 100000, 0.01, 0.0, nes)
	l2 := link.NewLink(s1, r1, 100000, 0.01, 0.0, nes)
	l3 := link.NewLink(s1, dns1, 100000, 0.01, 0.0, nes)
	l4 := link.NewLink(s1, dhcp1, 100000, 0.01, 0.0, nes)
	l5 := link.NewLink(n2, r1, 100000, 0.01, 0.0, nes)
	// s3 := nswitch.NewSwitch(nes, 7, "192.171.1.2", "00:1A:2B:3C:2D:5E")
	// s4 := nswitch.NewSwitch(nes, 8, "192.172.1.2", "00:1A:2B:3C:1D:6E")

	l1.PrintLink()
	l2.PrintLink()
	l3.PrintLink()
	l4.PrintLink()
	l5.PrintLink()

	dns1.AddDNSRecord("www.example.com", "192.168.2.1/24")

	n1.StartTraffic("www.example.com", 1.0, 50, 10000)

	// n2.SetTraffic("00:1A:2B:3C:4D:5E", 8000, 40.0, 10.0, 40.0, 85.0, 1.0)
	// linkを繋ぐ前
	// s1.PrintLinkStates()
	// s2.PrintLinkStates()
	nes.RunUntil(50.0)
	n1.PrintNode()
	n2.PrintNode()
	n1.PrintArpTable()
	n2.PrintArpTable()
	r1.PrintArpTable()
	// nes.PrintPacketLogs()
	nes.GenerateSummary()
	// s1.PrintForwadingTable()
	// s2.PrintForwadingTable()
	// linkを繋いだあと
	// s1.PrintLinkStates()
	// s2.PrintLinkStates()
	nes.Visualize()
}
