package e2e

import (
	"testing"

	"nt-simulator/link"
	"nt-simulator/node/host"
	"nt-simulator/node/router"
	"nt-simulator/nteventsched"
)

// TestRoutingTableLinearTopology は main.go と同じ直線トポロジーで，
// LSA 収束後に各ルーターのルーティングテーブルが正しく構築されることを検証する。
//
// トポロジー:
//
//	n1(1) -- r1(2) -- r2(3) -- r3(4) -- r4(5) -- n2(6)
//
// サブネット:
//
//	192.168.1.0/24  n1 <-> r1
//	10.1.2.0/24     r1 <-> r2
//	10.2.3.0/24     r2 <-> r3
//	10.3.4.0/24     r3 <-> r4
//	192.168.2.0/24  r4 <-> n2
func TestRoutingTableLinearTopology(t *testing.T) {
	nes := nteventsched.NewNtEventSched(false, false)

	n1 := host.NewHost(1, "00:1A:2B:3C:4D:5E", "192.168.1.1/24", 1500, nes)
	r1 := router.NewRouter(2, []string{"192.168.1.254/24", "10.1.2.1/24"}, nes)
	r2 := router.NewRouter(3, []string{"10.1.2.2/24", "10.2.3.2/24"}, nes)
	r3 := router.NewRouter(4, []string{"10.2.3.3/24", "10.3.4.3/24"}, nes)
	r4 := router.NewRouter(5, []string{"192.168.2.254/24", "10.3.4.4/24"}, nes)
	n2 := host.NewHost(6, "00:1A:2B:3C:4D:5F", "192.168.2.1/24", 1500, nes)

	link.NewLink(n1, r1, 100000, 0.001, 0.0, nes)
	link.NewLink(r1, r2, 200000, 0.001, 0.0, nes)
	link.NewLink(r2, r3, 100000, 0.001, 0.0, nes)
	link.NewLink(r3, r4, 200000, 0.001, 0.0, nes)
	link.NewLink(r4, n2, 100000, 0.001, 0.0, nes)

	// 初回 LSA は 0.3〜0.5s に送信され，直線 4 ホップのフラッディングは数 ms で完了するため 2s で十分収束する
	nes.RunUntil(2.0)

	// nexthop=-1 は直接接続（same network）を意味する
	tests := []struct {
		name   string
		r      interface{ GetRoutingEntries() map[string]int }
		want   map[string]int
	}{
		{
			name: "r1(ID=2)",
			r:    r1,
			want: map[string]int{
				"192.168.1.0/24": -1, // r1 自身のインターフェース
				"10.1.2.0/24":    -1, // r1-r2 間のサブネット（r2 と同じネットワーク）
				"10.2.3.0/24":    3,  // r2 経由
				"10.3.4.0/24":    3,  // r2 経由
				"192.168.2.0/24": 3,  // r2 経由
			},
		},
		{
			name: "r2(ID=3)",
			r:    r2,
			want: map[string]int{
				"192.168.1.0/24": 2,  // r1 経由
				"10.1.2.0/24":    -1, // r2 自身のインターフェース（r1 と同じネットワーク）
				"10.2.3.0/24":    -1, // r2 自身のインターフェース
				"10.3.4.0/24":    4,  // r3 経由
				"192.168.2.0/24": 4,  // r3 経由
			},
		},
		{
			name: "r3(ID=4)",
			r:    r3,
			want: map[string]int{
				"192.168.1.0/24": 3,  // r2 経由
				"10.1.2.0/24":    3,  // r2 経由
				"10.2.3.0/24":    -1, // r3 自身のインターフェース（r2 と同じネットワーク）
				"10.3.4.0/24":    -1, // r3 自身のインターフェース
				"192.168.2.0/24": 5,  // r4 経由
			},
		},
		{
			name: "r4(ID=5)",
			r:    r4,
			want: map[string]int{
				"192.168.1.0/24": 4,  // r3 経由
				"10.1.2.0/24":    4,  // r3 経由
				"10.2.3.0/24":    4,  // r3 経由
				"10.3.4.0/24":    -1, // r4 自身のインターフェース（r3 と同じネットワーク）
				"192.168.2.0/24": -1, // r4 自身のインターフェース
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.GetRoutingEntries()
			assertRoutingTable(t, got, tt.want)
		})
	}
}

// TestPacketDeliveryLinearTopology は main.go と同じ直線トポロジーで，
// LSA によるルーティングテーブル自動構築後に n1→n2 へパケットが正しく届くことを検証する。
func TestPacketDeliveryLinearTopology(t *testing.T) {
	nes := nteventsched.NewNtEventSched(false, false)

	n1 := host.NewHost(1, "00:1A:2B:3C:4D:5E", "192.168.1.1/24", 1500, nes)
	r1 := router.NewRouter(2, []string{"192.168.1.254/24", "10.1.2.1/24"}, nes)
	r2 := router.NewRouter(3, []string{"10.1.2.2/24", "10.2.3.2/24"}, nes)
	r3 := router.NewRouter(4, []string{"10.2.3.3/24", "10.3.4.3/24"}, nes)
	r4 := router.NewRouter(5, []string{"192.168.2.254/24", "10.3.4.4/24"}, nes)
	n2 := host.NewHost(6, "00:1A:2B:3C:4D:5F", "192.168.2.1/24", 1500, nes)

	link.NewLink(n1, r1, 100000, 0.001, 0.0, nes)
	link.NewLink(r1, r2, 200000, 0.001, 0.0, nes)
	link.NewLink(r2, r3, 100000, 0.001, 0.0, nes)
	link.NewLink(r3, r4, 200000, 0.001, 0.0, nes)
	link.NewLink(r4, n2, 100000, 0.001, 0.0, nes)

	// LSA は 0.3〜0.5s に送信されるので 2s で十分収束する。
	// 2.5s にパケット送信（10000byte → MTU(1500)-header(40)=1460byte/fragment → 7フラグメント）
	n1.SetTraffic("00:1A:2B:3C:4D:5F", "192.168.2.1/24", 8000, 2.5, 1.0, 40, 10000, 1.0)

	// 最遅リンク(100000bps)で 1500byte の送信に 0.12s かかる。
	// 7フラグメント × 5ホップで最大 ~4s。余裕を持って 15s まで実行する。
	nes.RunUntil(15.0)

	const wantFragments = 7 // ceil(10000 / (1500-40))
	if n2.ArrivedCount() != wantFragments {
		t.Errorf("n2 arrived fragments = %d, want %d", n2.ArrivedCount(), wantFragments)
	}
	const wantBytes = 10000
	if n2.ReceivedBytes() != wantBytes {
		t.Errorf("n2 received bytes = %d, want %d", n2.ReceivedBytes(), wantBytes)
	}
}

func assertRoutingTable(t *testing.T, got map[string]int, want map[string]int) {
	t.Helper()

	for network, wantNexthop := range want {
		gotNexthop, ok := got[network]
		if !ok {
			t.Errorf("ルート %s が存在しない (全エントリ: %v)", network, got)
			continue
		}
		if gotNexthop != wantNexthop {
			t.Errorf("ルート %s の nexthop = %d, want %d", network, gotNexthop, wantNexthop)
		}
	}

	if len(got) != len(want) {
		t.Errorf("エントリ数 = %d, want %d (余分なエントリ: %v)", len(got), len(want), got)
	}
}
