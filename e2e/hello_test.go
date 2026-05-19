package e2e

import (
	"sort"
	"testing"

	"nt-simulator/link"
	"nt-simulator/node/host"
	"nt-simulator/node/router"
	"nt-simulator/nteventsched"
)

// TestHelloExchange は，トポロジを構築してシミュレーションを回した後に
// 各ルーターが正しい隣接ルーターを発見できているかを検証する
func TestHelloExchange(t *testing.T) {
	nes := nteventsched.NewNtEventSched(false, false)

	n1 := host.NewHost(1, "00:1A:2B:3C:4D:5E", "192.168.1.1/24", 1500, nes)
	r1 := router.NewRouter(2, []string{"192.168.1.254/24", "10.1.3.1/24", "10.1.4.1/24"}, nes)
	r2 := router.NewRouter(3, []string{"192.168.2.254/24", "10.2.3.1/24", "10.2.4.1/24"}, nes)
	r3 := router.NewRouter(4, []string{"10.1.3.2/24", "10.2.3.2/24"}, nes)
	r4 := router.NewRouter(5, []string{"10.1.4.2/24", "10.2.4.2/24"}, nes)
	n2 := host.NewHost(6, "00:1A:2B:3C:4D:5F", "192.168.2.1/24", 1500, nes)

	link.NewLink(n1, r1, 100000, 0.001, 0.0, nes)
	link.NewLink(r2, n2, 100000, 0.001, 0.0, nes)
	link.NewLink(r1, r3, 200000, 0.001, 0.0, nes)
	link.NewLink(r1, r4, 100000, 0.001, 0.0, nes)
	link.NewLink(r2, r3, 200000, 0.001, 0.0, nes)
	link.NewLink(r2, r4, 100000, 0.001, 0.0, nes)

	// helloInterval=10.0 なので 15 秒あれば最初の hello が届く
	nes.RunUntil(15.0)

	// 直接接続しているルーター同士が隣接として登録されているか確認
	// n1/n2 はホストなので hello を送らない → ルーターの neighbors には現れない
	assertNeighbors(t, "r1", r1.GetNeighborIds(), []int{4, 5}) // r3, r4
	assertNeighbors(t, "r2", r2.GetNeighborIds(), []int{4, 5}) // r3, r4
	assertNeighbors(t, "r3", r3.GetNeighborIds(), []int{2, 3}) // r1, r2
	assertNeighbors(t, "r4", r4.GetNeighborIds(), []int{2, 3}) // r1, r2
}

func assertNeighbors(t *testing.T, routerName string, got, want []int) {
	t.Helper()
	sort.Ints(got)
	sort.Ints(want)
	if len(got) != len(want) {
		t.Errorf("%s: neighbors = %v, want %v", routerName, got, want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s: neighbors = %v, want %v", routerName, got, want)
			return
		}
	}
}
