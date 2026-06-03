package e2e

import (
	"testing"

	"nt-simulator/link"
	"nt-simulator/node/host"
	"nt-simulator/node/nswitch"
	"nt-simulator/nteventsched"
)

// TestFourHostSwitchRing は 4ホスト + 1スイッチ構成で
// ARP 自動解決後にリング状のトラフィックが正しく届くことを検証する
func TestFourHostSwitchRing(t *testing.T) {
	nes := nteventsched.NewNtEventSched(false, false)

	n1 := host.NewHost(1, "192.168.1.1/24", 1500, nes)
	n2 := host.NewHost(2, "192.168.1.2/24", 1500, nes)
	n3 := host.NewHost(3, "192.168.1.3/24", 1500, nes)
	n4 := host.NewHost(4, "192.168.1.4/24", 1500, nes)
	s1 := nswitch.NewSwitch(nes, 5, "192.168.1.11/24")

	link.NewLink(n1, s1, 100000, 0.01, 0.0, nes)
	link.NewLink(n2, s1, 200000, 0.01, 0.0, nes)
	link.NewLink(n3, s1, 200000, 0.01, 0.0, nes)
	link.NewLink(n4, s1, 200000, 0.01, 0.0, nes)

	n1.SetTraffic(n2.IpAddress, 10000, 1.0, 10.0, 40, 10000, 1.0)
	n2.SetTraffic(n3.IpAddress, 10000, 1.0, 10.0, 40, 10000, 1.0)
	n3.SetTraffic(n4.IpAddress, 10000, 1.0, 10.0, 40, 10000, 1.0)
	n4.SetTraffic(n1.IpAddress, 10000, 1.0, 10.0, 40, 10000, 1.0)

	nes.RunUntil(50.0)

	const wantFragments = 7 // ceil(10000 / (1500-40))
	const wantBytes = 10000 // SetTraffic で指定した payloadSize

	if n1.ArrivedCount() != wantFragments {
		t.Errorf("n1 arrived fragments = %d, want %d", n1.ArrivedCount(), wantFragments)
	}
	if n1.ReceivedBytes() != wantBytes {
		t.Errorf("n1 received bytes = %d, want %d", n1.ReceivedBytes(), wantBytes)
	}
	if n2.ArrivedCount() != wantFragments {
		t.Errorf("n2 arrived fragments = %d, want %d", n2.ArrivedCount(), wantFragments)
	}
	if n2.ReceivedBytes() != wantBytes {
		t.Errorf("n2 received bytes = %d, want %d", n2.ReceivedBytes(), wantBytes)
	}
	if n3.ArrivedCount() != wantFragments {
		t.Errorf("n3 arrived fragments = %d, want %d", n3.ArrivedCount(), wantFragments)
	}
	if n3.ReceivedBytes() != wantBytes {
		t.Errorf("n3 received bytes = %d, want %d", n3.ReceivedBytes(), wantBytes)
	}
	if n4.ArrivedCount() != wantFragments {
		t.Errorf("n4 arrived fragments = %d, want %d", n4.ArrivedCount(), wantFragments)
	}
	if n4.ReceivedBytes() != wantBytes {
		t.Errorf("n4 received bytes = %d, want %d", n4.ReceivedBytes(), wantBytes)
	}
}
