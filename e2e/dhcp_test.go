package e2e

import (
	"testing"

	"nt-simulator/address"
	"nt-simulator/link"
	"nt-simulator/node/dhcpserver"
	"nt-simulator/node/dnsserver"
	"nt-simulator/node/host"
	"nt-simulator/node/nswitch"
	"nt-simulator/node/router"
	"nt-simulator/nteventsched"
)

// TestMainTopologyWithDHCP は main.go と同じ設定で
// DHCP による IP 割り当てと DNS 解決後のパケット配送を検証する。
//
// トポロジ: n1 -- s1 -- r1 -- n2
//
//	s1 -- dns1, dhcp1
func TestMainTopologyWithDHCP(t *testing.T) {
	nes := nteventsched.NewNtEventSched(false, false)

	n1 := host.NewHost(1, "192.168.1.0/24", 1500, nes)
	n2 := host.NewHost(2, "192.168.2.1/24", 1500, nes)
	r1 := router.NewRouter(3, []string{"192.168.1.254/24", "192.168.2.254/24"}, nes)
	dns1 := dnsserver.NewDNSServer(nes, 4, "192.168.1.200/24")
	s1 := nswitch.NewSwitch(nes, 5, "192.168.1.240/24")
	dhcp1 := dhcpserver.NewDHCPServer(nes, 6, "192.168.1.250/24", "192.168.1.200/24", "192.168.1.0/24")
	dhcp1.ReserveIP(s1.GetIPAddresses()[0].String())
	dhcp1.ReserveIP(dns1.GetIPAddresses()[0].String())
	for _, ip := range r1.GetIPAddresses() {
		dhcp1.ReserveIP(ip.String())
	}
	dhcp1.ReserveIP("192.168.1.250/24")

	link.NewLink(n1, s1, 100000, 0.01, 0.0, nes)
	link.NewLink(s1, r1, 100000, 0.01, 0.0, nes)
	link.NewLink(s1, dns1, 100000, 0.01, 0.0, nes)
	link.NewLink(s1, dhcp1, 100000, 0.01, 0.0, nes)
	link.NewLink(n2, r1, 100000, 0.01, 0.0, nes)

	const (
		domain      = "www.example.com"
		startTime   = 1.0
		headerSize  = 50
		payloadSize = 10000
	)

	dns1.AddDNSRecord(domain, "192.168.2.1/24")
	n1.StartTraffic(domain, startTime, headerSize, payloadSize, "UDP")

	nes.RunUntil(50.0)

	assignedIP := n1.GetIPAddresses()[0]
	if assignedIP.IsNetworkAddress() {
		t.Errorf("n1 IP = %s, want DHCP-assigned host address (not network address)", assignedIP)
	}

	reservedIPs := map[string]bool{
		"192.168.1.240/24": true,
		"192.168.1.200/24": true,
		"192.168.1.254/24": true,
		"192.168.1.250/24": true,
	}
	if reservedIPs[assignedIP.String()] {
		t.Errorf("n1 was assigned reserved IP %s", assignedIP)
	}

	network := address.NewIPAddress("192.168.1.0/24")
	if !assignedIP.IsSameNetwork(network) {
		t.Errorf("n1 IP = %s, want address in 192.168.1.0/24", assignedIP)
	}

	const wantFragments = 7 // ceil(10000 / (1500-50))
	if n2.ArrivedCount() != wantFragments {
		t.Errorf("n2 arrived fragments = %d, want %d", n2.ArrivedCount(), wantFragments)
	}
	if n2.ReceivedBytes() != payloadSize {
		t.Errorf("n2 received bytes = %d, want %d", n2.ReceivedBytes(), payloadSize)
	}
}
