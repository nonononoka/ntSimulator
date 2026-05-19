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

type neighborInfo struct {
	lastHelloTime float64
	link          *link.Link
	info          packet.HellopPayload
}

type topologyEntry struct {
	sequenceNumber int
	linkStateInfos map[string]packet.LinkStateInfo // linkごとのstate情報
}

type router struct {
	*basenode.BaseNode
	availableIps          map[*address.IpAddress]bool       // CIDR表記のIPアドレスを辞書に変換して，使用状況を記録
	interfaces            map[*link.Link]*address.IpAddress // リンクとIPアドレスのマッピング
	routingTable          map[*address.IpAddress]entry
	helloInterval         float64
	neighbors             map[int]*neighborInfo
	lsaSequenceNumber     int
	lsaInterval           float64
	isTopologyInitialized bool
	topologyDatabase      map[int]topologyEntry // 各router idのリンク情報を管理
}

func NewRouter(nodeId int, ipAddreses []string, nes *nteventsched.NtEventSched) *router {
	availableIps := make(map[*address.IpAddress]bool)
	for _, ip := range ipAddreses {
		availableIps[address.NewIPAddress(ip)] = false
	}
	r := &router{
		BaseNode:              basenode.NewBaseNode(nodeId, nes),
		availableIps:          availableIps,
		interfaces:            make(map[*link.Link]*address.IpAddress),
		routingTable:          make(map[*address.IpAddress]entry),
		helloInterval:         10.0,
		neighbors:             make(map[int]*neighborInfo),
		lsaInterval:           10.0,
		isTopologyInitialized: false,
		topologyDatabase:      make(map[int]topologyEntry), // ルーターIDごとのLsaの情報を管理
	}
	nes.AddNode(r)
	r.scheduleHelloPacket()
	r.scheduleLsaPacket()
	return r
}

func (s *router) NodeColor() string { return "blue" }

// リーターに新しいリンクを追加する
func (r *router) AddLink(link *link.Link, ipAddress *address.IpAddress) {
	if _, ok := r.interfaces[link]; !ok {
		r.interfaces[link] = ipAddress
		r.markIpAsUsed(ipAddress)
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

func (r *router) ReceivePacket(p packet.PacketI, receivedLink *link.Link) {
	if helloP, ok := p.(*packet.HelloP); ok {
		r.GetNES().LogPacketInfo(p, "router hello received", r.NodeId())
		r.receiveHelloPacket(helloP, receivedLink)
		return
	}

	if lsaP, ok := p.(*packet.LsaP); ok {
		r.GetNES().LogPacketInfo(p, "router lsa received", r.NodeId())
		r.receiveLsaPacket(lsaP, receivedLink)
		return
	}

	p.DecrementTTL()
	if p.GetTTL() <= 0 {
		r.GetNES().LogPacketInfo(p, "dropped due to TTL expired", r.NodeId())
		return
	}

	r.GetNES().LogPacketInfo(p, "router received", r.NodeId())

	// 普通のパケットの処理
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
	for ad, used := range r.availableIps {
		if !used {
			addresses = append(addresses, ad)
		}
	}
	return addresses
}

func (r *router) GetTopologyRouterIds() []int {
	ids := make([]int, 0, len(r.topologyDatabase))
	for id := range r.topologyDatabase {
		ids = append(ids, id)
	}
	return ids
}

func (r *router) GetTopologyLinkCount(routerId int) (int, bool) {
	entry, ok := r.topologyDatabase[routerId]
	if !ok {
		return 0, false
	}
	return len(entry.linkStateInfos), true
}

func (r *router) GetNeighborIds() []int {
	ids := make([]int, 0, len(r.neighbors))
	for id := range r.neighbors {
		ids = append(ids, id)
	}
	return ids
}

func (r *router) markIpAsUsed(ipAddress *address.IpAddress) {
	if _, ok := r.availableIps[ipAddress]; ok {
		r.availableIps[ipAddress] = true
	} else {
		panic("IPアドレスはこのルータに存在しません")
	}
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

	// 宛先がOSPFのマルチキャストアドレスの場合
	if destinationAddress.String() == "224.0.0.5/32" {
		for l := range r.interfaces { // 全部のinterfaceにパケットを送信
			l.EnqueuePacket(p, r)
		}
	} else if ok { // ルーティング成功
		r.GetNES().LogPacketInfo(p, "router forwarded", r.NodeId())
		entry.link.EnqueuePacket(p, r)
	} else {
		r.GetNES().LogPacketInfo(p, "router dropped", r.NodeId())
	}
	// TODO：デフォルトルートの実装
}
