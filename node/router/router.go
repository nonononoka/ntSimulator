package router

import (
	"nt-simulator/address"
	"nt-simulator/link"
	"nt-simulator/node/basenode"
	"nt-simulator/nteventsched"
	"nt-simulator/packet"
)

/*
[ルーターA]----リンク----[ルーターB]
 10.0.0.1                10.0.0.2

ルーターA：
interfaces → このリンクに対して 10.0.0.1（自分のアドレス）
nexthop → 10.0.0.2（ルーターBのアドレス）
*/

type entry struct {
	nexthop *address.IpAddress
	link    *link.Link
}

type router struct {
	*basenode.BaseNode
	availableIps map[*address.IpAddress]bool       // CIDR表記のIPアドレスを辞書に変換して，使用状況を記録
	interfaces   map[*link.Link]*address.IpAddress // リンクとIPアドレスのマッピング
	routingTable map[*address.IpAddress]entry
}

func NewRouter(nodeId int, ipAddreses []string, nes *nteventsched.NtEventSched) *router {
	availableIps := make(map[*address.IpAddress]bool)
	for _, ip := range ipAddreses {
		availableIps[address.NewIPAddress(ip)] = false
	}
	r := &router{
		BaseNode:     basenode.NewBaseNode(nodeId, nes),
		availableIps: availableIps,
		interfaces:   make(map[*link.Link]*address.IpAddress),
		routingTable: make(map[*address.IpAddress]entry),
	}
	nes.AddNode(r)
	return r
}

func (s *router) NodeColor() string { return "blue" }

// リーターに新しいリンクを追加する
func (r *router) AddLink(link *link.Link, ipAddress *address.IpAddress) {
	if _, ok := r.interfaces[link]; !ok {
		r.interfaces[link] = ipAddress
	}
}

// ルーティングテーブルにルートを追加するメソッド
// このdestinationCIDRはネットワークアドレス
func (r *router) AddRoute(destinationCIDR string, nexthop string, link *link.Link) {
	var nextHopAddr *address.IpAddress
	if nexthop != "" {
		nextHopAddr = address.NewIPAddress(nexthop)
	}
	r.routingTable[address.NewIPAddress(destinationCIDR)] = entry{nexthop: nextHopAddr, link: link}
}

func (r *router) markIpAsUsed(ipAddress *address.IpAddress) {
	if _, ok := r.availableIps[ipAddress]; ok {
		r.availableIps[ipAddress] = true
	} else {
		panic("IPアドレスはこのルータに存在しません")
	}
}

func (r *router) getAvailableIpAddresses() []*address.IpAddress {
	availableIps := make([]*address.IpAddress, 10)
	for ip, used := range r.availableIps {
		if !used {
			availableIps = append(availableIps, ip)
		}
	}
	return availableIps
}

// CIDR形式のルーティングテーブルから宛先IPアドレスに最適なルートを検索する
// destinationIPが，特定のネットワークに属するならそこが最適ルートとする
func (r *router) getRoute(destionationIp *address.IpAddress) (entry, bool) {
	for networkCIDR, next := range r.routingTable {
		if networkCIDR.IsSameNetwork(destionationIp) {
			return next, true
		}
	}
	return entry{}, false // ルートが見つからなかった場合
}

func (r *router) forwardPacket(p packet.PacketI) {
	destinationAddress := p.GetHeader().DestIp
	entry, ok := r.getRoute(destinationAddress) // このdestination Addressに行くなら，このlinkを辿ってこのIPアドレスに行け
	// 直接接続の場合は，nexthopはnilになっている．
	if ok {
		r.GetNES().LogPacketInfo(p, "router forwarded", r.NodeId())
		entry.link.EnqueuePacket(p, r)
	} else {
		r.GetNES().LogPacketInfo(p, "router dropped", r.NodeId())
	}
	// TODO：デフォルトルートの実装
}

func (r *router) ReceivePacket(p packet.PacketI, receivedLink *link.Link) {
	r.GetNES().LogPacketInfo(p, "router received", r.NodeId())
	destIp := p.GetHeader().DestIp
	for _, interfaceCIDR := range r.interfaces {
		if destIp.IsSameNetwork(interfaceCIDR) {
			if destIp.String() == interfaceCIDR.String() { // パケットがルーター宛
				r.GetNES().LogPacketInfo(p, "arrived to router", r.NodeId())
			} else { // ルーター宛ではないが，そのルーターと同じネットワークだった場合
				r.forwardPacket(p)
			}
			return
		}
	}
	r.forwardPacket(p)
}

func (r *router) GetIPAddresses() []*address.IpAddress {
	var addresses []*address.IpAddress
	for ad, _ := range r.availableIps {
		addresses = append(addresses, ad)
	}
	return addresses
}
