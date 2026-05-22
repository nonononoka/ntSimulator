package e2e

import (
	"testing"

	"nt-simulator/link"
	"nt-simulator/node/host"
	"nt-simulator/node/router"
	"nt-simulator/nteventsched"
)

// TestPacketDelivery は n1→r1→r2→n2 の経路でパケットが届くかを検証する
func TestPacketDelivery(t *testing.T) {
	nes := nteventsched.NewNtEventSched(false, false)

	n1 := host.NewHost(1, "192.168.1.1/24", 1500, nes)
	r1 := router.NewRouter(2, []string{"192.168.1.254/24", "10.0.0.1/24"}, nes)
	r2 := router.NewRouter(3, []string{"192.168.2.254/24", "10.0.0.2/24"}, nes)
	n2 := host.NewHost(4, "192.168.2.1/24", 1500, nes)

	l1 := link.NewLink(n1, r1, 100000, 0.001, 0.0, nes)
	l2 := link.NewLink(r1, r2, 100000, 0.001, 0.0, nes)
	l3 := link.NewLink(n2, r2, 100000, 0.001, 0.0, nes)

	r1.AddRoute("192.168.1.0/24", -1, l1)
	r1.AddRoute("192.168.2.0/24", 3, l2)
	r2.AddRoute("192.168.2.0/24", -1, l3)
	r2.AddRoute("192.168.1.0/24", 2, l2)

	// startTime=1.0 でパケット送信，payload=10000byte → MTU(1500)-header(40)=1460byte/fragment → 7フラグメント
	n1.SetTraffic("192.168.2.1/24", 8000, 1.0, 10.0, 40, 10000, 1.0)

	nes.RunUntil(10.0)

	const wantFragments = 7 // ceil(10000 / (1500-40))
	if n2.ArrivedCount() != wantFragments {
		t.Errorf("n2 arrived fragments = %d, want %d", n2.ArrivedCount(), wantFragments)
	}
	const wantBytes = 10000 // SetTraffic で指定した payloadSize
	if n2.ReceivedBytes() != wantBytes {
		t.Errorf("n2 received bytes = %d, want %d", n2.ReceivedBytes(), wantBytes)
	}
}
