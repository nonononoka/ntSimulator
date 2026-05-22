package e2e

import (
	"sort"
	"testing"

	"nt-simulator/link"
	"nt-simulator/node/host"
	"nt-simulator/node/router"
	"nt-simulator/nteventsched"
)

// TestLSATopologyExchange は，LSAパケットのフラッディングによって
// 各ルーターがネットワーク全体のトポロジー情報を取得できることを検証する
//
// トポロジー（main.go と同じ構成）:
//
//	n1 -- r1 -- r3 -- r2 -- n2
//	       \   /  \   /
//	        \ /    \ /
//	         r4----（r3-r4リンクなし，r1-r4 と r2-r4）
func TestLSATopologyExchange(t *testing.T) {
	nes := nteventsched.NewNtEventSched(false, false)

	n1 := host.NewHost(1, "192.168.1.1/24", 1500, nes)
	r1 := router.NewRouter(2, []string{"192.168.1.254/24", "10.1.3.1/24", "10.1.4.1/24"}, nes)
	r2 := router.NewRouter(3, []string{"192.168.2.254/24", "10.2.3.1/24", "10.2.4.1/24"}, nes)
	r3 := router.NewRouter(4, []string{"10.1.3.2/24", "10.2.3.2/24"}, nes)
	r4 := router.NewRouter(5, []string{"10.1.4.2/24", "10.2.4.2/24"}, nes)
	n2 := host.NewHost(6, "192.168.2.1/24", 1500, nes)

	link.NewLink(n1, r1, 100000, 0.001, 0.0, nes)
	link.NewLink(r2, n2, 100000, 0.001, 0.0, nes)
	link.NewLink(r1, r3, 200000, 0.001, 0.0, nes)
	link.NewLink(r1, r4, 100000, 0.001, 0.0, nes)
	link.NewLink(r2, r3, 200000, 0.001, 0.0, nes)
	link.NewLink(r2, r4, 100000, 0.001, 0.0, nes)

	// 最初のLSAは0.3〜0.5sに送信される．フラッディングの伝播は数msなので2sで十分収束する
	nes.RunUntil(2.0)

	allRouterIds := []int{2, 3, 4, 5} // r1, r2, r3, r4 の nodeId

	// 各ルーターのトポロジーDBに全ルーターの情報が入っているか確認
	assertTopologyRouterIds(t, "r1", r1.GetTopologyRouterIds(), allRouterIds)
	assertTopologyRouterIds(t, "r2", r2.GetTopologyRouterIds(), allRouterIds)
	assertTopologyRouterIds(t, "r3", r3.GetTopologyRouterIds(), allRouterIds)
	assertTopologyRouterIds(t, "r4", r4.GetTopologyRouterIds(), allRouterIds)

	// 各ルーターのエントリが持つリンク数も検証する
	// r1(node2): n1, r3, r4 への3リンク
	// r2(node3): n2, r3, r4 への3リンク
	// r3(node4): r1, r2 への2リンク
	// r4(node5): r1, r2 への2リンク
	wantLinkCounts := map[int]int{2: 3, 3: 3, 4: 2, 5: 2}
	for _, r := range [...]interface {
		GetTopologyLinkCount(int) (int, bool)
		GetTopologyRouterIds() []int
	}{r1, r2, r3, r4} {
		for routerId, wantCount := range wantLinkCounts {
			got, ok := r.GetTopologyLinkCount(routerId)
			if !ok {
				continue // すでに上のassertで検出済み
			}
			if got != wantCount {
				t.Errorf("topology entry for router %d: link count = %d, want %d", routerId, got, wantCount)
			}
		}
	}
}

func assertTopologyRouterIds(t *testing.T, routerName string, got, want []int) {
	t.Helper()
	sort.Ints(got)
	sort.Ints(want)
	if len(got) != len(want) {
		t.Errorf("%s topology DB: router IDs = %v, want %v", routerName, got, want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s topology DB: router IDs = %v, want %v", routerName, got, want)
			return
		}
	}
}
