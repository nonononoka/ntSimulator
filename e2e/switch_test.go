package e2e

import (
	"testing"

	"nt-simulator/link"
	"nt-simulator/node/host"
	"nt-simulator/node/nswitch"
	"nt-simulator/nteventsched"
)

// TestSwitchDelivery は n1→s1→n2 の構成でスイッチ経由のパケット配送を検証する
func TestSwitchDelivery(t *testing.T) {
	nes := nteventsched.NewNtEventSched(false, false)

	n1 := host.NewHost(1, "192.168.1.1/24", 1500, nes)
	n2 := host.NewHost(2, "192.168.1.2/24", 1500, nes)
	s1 := nswitch.NewSwitch(nes, 3, "192.168.1.3/24")

	link.NewLink(s1, n1, 100000, 0.001, 0.0, nes)
	link.NewLink(s1, n2, 100000, 0.001, 0.0, nes)

	// headerSize=20, payloadSize=1000 → 1020 byte < MTU(1500) なのでフラグメントなし
	n1.SetTraffic(n2.IpAddress, 8000, 1.0, 10.0, 20, 1000, 1.0)

	nes.RunUntil(5.0)

	const wantBytes = 1000
	if n2.ReceivedBytes() != wantBytes {
		t.Errorf("n2 received bytes = %d, want %d", n2.ReceivedBytes(), wantBytes)
	}
}
