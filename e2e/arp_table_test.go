package e2e

// arpテーブルを適切に設定すれば、パケットが届くことのテスト
// arpプロトコルのテストではない

import (
	"testing"

	"nt-simulator/link"
	"nt-simulator/node/host"
	"nt-simulator/node/nswitch"
	"nt-simulator/node/router"
	"nt-simulator/nteventsched"
)

// TestSwitchDelivery は n1→s1→n2 の構成でスイッチ経由のパケット配送を検証する
func TestArpTableSetting(t *testing.T) {
	nes := nteventsched.NewNtEventSched(true, true)
	n1 := host.NewHost(1, "192.168.1.1/24", 1500, nes)
	n2 := host.NewHost(2, "192.168.2.1/24", 1500, nes)
	r1 := router.NewRouter(3, []string{"192.168.1.254/24", "10.1.1.1/24"}, nes)
	r2 := router.NewRouter(4, []string{"192.168.2.254/24", "10.1.1.2/24"}, nes)
	s1 := nswitch.NewSwitch(nes, 5, "192.168.1.11/24")
	s2 := nswitch.NewSwitch(nes, 6, "192.168.2.11/24")

	link.NewLink(n1, s1, 100000, 0.01, 0.0, nes)
	l2 := link.NewLink(s1, r1, 100000, 0.01, 0.0, nes)
	l3 := link.NewLink(r1, r2, 200000, 0.01, 0.0, nes)
	l4 := link.NewLink(r2, s2, 100000, 0.01, 0.0, nes)
	link.NewLink(s2, n2, 200000, 0.01, 0.0, nes)

	n1.AddToArpTable(n2.IpAddress, r1.GetMacAddress(l2))
	n2.AddToArpTable(n1.IpAddress, r2.GetMacAddress(l4))
	r1.AddToArpTable(n1.IpAddress, n1.MacAddress)
	r1.AddToArpTable(n2.IpAddress, r2.GetMacAddress(l3))
	r2.AddToArpTable(n1.IpAddress, r1.GetMacAddress(l3))
	r2.AddToArpTable(n2.IpAddress, n2.MacAddress)

	n1.StartTraffic(n2.IpAddress.String(), 1.0, 40, 10000)
	nes.RunUntil(50.0)

	const wantFragments = 7 // ceil(10000 / (1500-40))
	if n2.ArrivedCount() != wantFragments {
		t.Errorf("n2 arrived fragments = %d, want %d", n2.ArrivedCount(), wantFragments)
	}
	const wantBytes = 10000 // StartTraffic で指定した payloadSize
	if n2.ReceivedBytes() != wantBytes {
		t.Errorf("n2 received bytes = %d, want %d", n2.ReceivedBytes(), wantBytes)
	}
}
