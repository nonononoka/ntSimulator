package e2e

import (
	"testing"

	"nt-simulator/link"
	"nt-simulator/node/dnsserver"
	"nt-simulator/node/host"
	"nt-simulator/node/nswitch"
	"nt-simulator/node/router"
	"nt-simulator/nteventsched"
)

// TestMainTopologyPacketDelivery は main.go と同じ設定で
// DNS 解決後に n1 から n2 へパケットが届くことを検証する。
//
// トポロジ: n1 -- s1 -- r1 -- n2、DNS サーバー (dns1) は s1 経由で接続
func TestMainTopologyPacketDelivery(t *testing.T) {
	nes := nteventsched.NewNtEventSched(false, false)

	n1 := host.NewHost(1, "192.168.1.1/24", 1500, nes)
	n2 := host.NewHost(2, "192.168.2.1/24", 1500, nes)
	r1 := router.NewRouter(3, []string{"192.168.1.254/24", "192.168.2.254/24"}, nes)
	dns1 := dnsserver.NewDNSServer(nes, 4, "192.168.1.200/24")
	s1 := nswitch.NewSwitch(nes, 5, "192.168.1.11/24")

	link.NewLink(n1, s1, 100000, 0.01, 0.0, nes)
	link.NewLink(s1, r1, 100000, 0.01, 0.0, nes)
	link.NewLink(s1, dns1, 100000, 0.01, 0.0, nes)
	link.NewLink(n2, r1, 100000, 0.01, 0.0, nes)

	const (
		domain      = "www.example.com"
		startTime   = 1.0
		headerSize  = 50
		payloadSize = 10000
	)

	dns1.AddDNSRecord(domain, "192.168.2.1/24")
	n1.StartTraffic(domain, startTime, headerSize, payloadSize)

	nes.RunUntil(50.0)

	const wantFragments = 7 // ceil(10000 / (1500-50))
	if n2.ArrivedCount() != wantFragments {
		t.Errorf("n2 arrived fragments = %d, want %d", n2.ArrivedCount(), wantFragments)
	}
	if n2.ReceivedBytes() != payloadSize {
		t.Errorf("n2 received bytes = %d, want %d", n2.ReceivedBytes(), payloadSize)
	}
}
