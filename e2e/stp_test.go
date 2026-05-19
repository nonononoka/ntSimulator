package e2e

import (
	"testing"

	"nt-simulator/link"
	"nt-simulator/node/host"
	"nt-simulator/node/nswitch"
	"nt-simulator/nteventsched"
)

// TestSTPTreeFormation は，4スイッチのフルメッシュ構成でSTPが正しくスパニングツリーを
// 形成するかを検証する．s1(nodeId=5)が最小IDなのでルートブリッジになり，
// s1-s2, s1-s3, s1-s4 がforwarding, その他のスイッチ間リンクはblockingになる．
func TestSTPTreeFormation(t *testing.T) {
	nes := nteventsched.NewNtEventSched(false, false)

	n1 := host.NewHost(1, "00:1A:2B:3C:4D:5E", "10.0.0.1/24", 1500, nes)
	n2 := host.NewHost(2, "00:1A:2B:3C:4D:5F", "10.0.0.2/24", 1500, nes)
	s1 := nswitch.NewSwitch(nes, 5, "10.0.0.5/24", "00:00:00:00:00:05")
	s2 := nswitch.NewSwitch(nes, 6, "10.0.0.6/24", "00:00:00:00:00:06")
	s3 := nswitch.NewSwitch(nes, 7, "10.0.0.7/24", "00:00:00:00:00:07")
	s4 := nswitch.NewSwitch(nes, 8, "10.0.0.8/24", "00:00:00:00:00:08")

	link.NewLink(s1, n1, 100000, 0.001, 0.0, nes)
	link.NewLink(s3, n2, 100000, 0.001, 0.0, nes)
	ls1s2 := link.NewLink(s1, s2, 100000, 0.001, 0.0, nes)
	ls1s3 := link.NewLink(s1, s3, 100000, 0.001, 0.0, nes)
	ls1s4 := link.NewLink(s1, s4, 100000, 0.001, 0.0, nes)
	ls2s3 := link.NewLink(s2, s3, 100000, 0.001, 0.0, nes)
	ls2s4 := link.NewLink(s2, s4, 100000, 0.001, 0.0, nes)
	ls3s4 := link.NewLink(s3, s4, 100000, 0.001, 0.0, nes)

	nes.RunUntil(0.5)

	// s1 はルートブリッジ: スイッチ間の全ポートが forwarding
	assertLinkState(t, "s1→s2", s1.GetLinkState(ls1s2), "forwarding")
	assertLinkState(t, "s1→s3", s1.GetLinkState(ls1s3), "forwarding")
	assertLinkState(t, "s1→s4", s1.GetLinkState(ls1s4), "forwarding")

	// s2: s1へのルートポートが forwarding，冗長リンクは blocking
	assertLinkState(t, "s2→s1", s2.GetLinkState(ls1s2), "forwarding")
	assertLinkState(t, "s2→s3", s2.GetLinkState(ls2s3), "blocking")
	assertLinkState(t, "s2→s4", s2.GetLinkState(ls2s4), "blocking")

	// s3: s1へのルートポートが forwarding，冗長リンクは blocking
	assertLinkState(t, "s3→s1", s3.GetLinkState(ls1s3), "forwarding")
	assertLinkState(t, "s3→s2", s3.GetLinkState(ls2s3), "blocking")
	assertLinkState(t, "s3→s4", s3.GetLinkState(ls3s4), "blocking")

	// s4: s1へのルートポートが forwarding，冗長リンクは blocking
	assertLinkState(t, "s4→s1", s4.GetLinkState(ls1s4), "forwarding")
	assertLinkState(t, "s4→s2", s4.GetLinkState(ls2s4), "blocking")
	assertLinkState(t, "s4→s3", s4.GetLinkState(ls3s4), "blocking")
}

func assertLinkState(t *testing.T, label, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: state = %q, want %q", label, got, want)
	}
}
