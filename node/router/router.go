package router

import (
	"fmt"
	"math/rand/v2"
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

func (r *router) scheduleHelloPacket() {
	randomDelay := rand.Float64() * 0.1
	r.GetNES().ScheduleEvent(r.GetNES().CurrentTime+randomDelay, func(args ...any) { r.sendHelloPacket() })
}

func (r *router) sendHelloPacket() {
	var neighbors []int
	for neighborRouterId := range r.neighbors {
		neighbors = append(neighbors, neighborRouterId)
	}

	for l, interfaceCIDR := range r.interfaces {
		helloP := packet.NewHelloP(address.NewMacAddress("00:00:00:00:00:00"), interfaceCIDR, r.GetNES().CurrentTime, r.NodeId(), r.helloInterval, neighbors)
		l.EnqueuePacket(helloP, r)
	}
	r.GetNES().ScheduleEvent(r.GetNES().CurrentTime+r.helloInterval, func(args ...any) {
		r.sendHelloPacket()
	})
}

func (r *router) scheduleLsaPacket() {
	randomDelay := 0.3 + rand.Float64()*0.2
	r.GetNES().ScheduleEvent(r.GetNES().CurrentTime+randomDelay, func(args ...any) { r.sendLsaPacket() })
}

func (r *router) sendLsaPacket() {
	seqNumber := r.incrementLsaSequence()
	linkStateInfos := r.getLinkStateInfos()

	for link, ipAddress := range r.interfaces {
		lsaP := packet.NewLsaP(
			address.NewMacAddress("00:00:00:00:00:00"), ipAddress, r.GetNES().CurrentTime, r.NodeId(), seqNumber, linkStateInfos)
		link.EnqueuePacket(lsaP, r)
	}

	r.GetNES().ScheduleEvent(r.GetNES().CurrentTime+r.lsaInterval, func(args ...any) {
		r.sendLsaPacket()
	})
}

func (r *router) getLinkStateInfos() map[string]packet.LinkStateInfo {
	linkStateInfos := make(map[string]packet.LinkStateInfo)
	for l, ipAddress := range r.interfaces {
		linkStateInfos[l.GetId()] = packet.LinkStateInfo{IpAddress: ipAddress.String(), Cost: l.GetLinkCost()}
	}
	return linkStateInfos
}

func (r *router) incrementLsaSequence() int {
	r.lsaSequenceNumber += 1
	return r.lsaSequenceNumber
}

func (r *router) floodLsa(p *packet.LsaP) {
	// 受信したLSAパケットを他のルーターにフラッドする
	lp, err := p.ParsePayload()
	if err != nil {
		fmt.Printf("Lsa parse error: %v\n", err)
		return
	}
	routerId := lp.RouterId

	for link := range r.interfaces {
		if link.NodeX().NodeId() != routerId && link.NodeY().NodeId() != routerId {
			link.EnqueuePacket(p, r)
		}
	}
}

func (r *router) receiveHelloPacket(p *packet.HelloP, receivedLink *link.Link) {
	hp, err := p.ParsePayload()
	if err != nil {
		fmt.Printf("Hello parse error: %v\n", err)
		return
	}
	routerId := hp.RouterId
	newNeighbor := false
	now := r.GetNES().CurrentTime

	if _, ok := r.neighbors[routerId]; !ok {
		// 未知のルーターなので、新しい隣接情報を更新
		newNeighbor = true
		r.neighbors[routerId] = &neighborInfo{
			lastHelloTime: now,
			link:          receivedLink,
			info:          hp,
		}
	} else {
		// 既知のルーターの隣接情報を更新
		lastHelloTime := r.neighbors[routerId].lastHelloTime
		if now > lastHelloTime {
			r.neighbors[routerId].lastHelloTime = now
		}
		if receivedLink != r.neighbors[routerId].link {
			newNeighbor = true
			r.neighbors[routerId].link = receivedLink
		}
		if !hp.Equals(r.neighbors[routerId].info) {
			newNeighbor = true
			r.neighbors[routerId].info = hp
		}
	}
	if r.GetNES().Verbose {

		if newNeighbor {
			r.printNeighborInfo()
		} else {
			fmt.Printf("%v Helloパケットを受信しましたが、隣接ルーターの情報は更新されていません。ルーターID: %v \n", r.GetNES().CurrentTime, r.NodeId())
		}
	}
}

func (r *router) receiveLsaPacket(p *packet.LsaP, receivedLink *link.Link) {
	lp, err := p.ParsePayload()
	if err != nil {
		fmt.Printf("Lsa parse error: %v\n", err)
		return
	}

	if !r.isTopologyInitialized {
		r.isTopologyInitialized = true
		r.initializeTopologyDatabase()
	}

	routerId := lp.RouterId
	lsaInfo := lp.LinkStateInfos
	seqNumber := lp.SequenceNumber
	now := r.GetNES().CurrentTime

	var currentLsaInfo topologyEntry
	if _, ok := r.topologyDatabase[routerId]; !ok {
		currentLsaInfo = topologyEntry{}
	} else {
		currentLsaInfo = r.topologyDatabase[routerId]
	}

	// 受信したLSAが新しい情報を持っている場合、トポロジーデータベースを更新
	if seqNumber > currentLsaInfo.sequenceNumber {
		r.topologyDatabase[routerId] = topologyEntry{
			sequenceNumber: seqNumber,
			linkStateInfos: lsaInfo,
		}
		if r.GetNES().Verbose {
			r.printTopologyDatabase()
		}
		r.floodLsa(p)
	} else {
		fmt.Printf("%v 古いLSAを受信しました %v\n", now, r.NodeId())
	}
}

func (r *router) initializeTopologyDatabase() {
	linkStateInfos := make(map[string]packet.LinkStateInfo)
	for link, ipAddress := range r.interfaces {
		linkStateInfos[link.GetId()] = packet.LinkStateInfo{IpAddress: ipAddress.String(), Cost: link.GetLinkCost()}
	}

	r.topologyDatabase[r.NodeId()] = topologyEntry{sequenceNumber: 0, linkStateInfos: linkStateInfos}
}

func (r *router) printNeighborInfo() {
	for routerId, neighborInfo := range r.neighbors {
		fmt.Printf("ルーターID: %v \n", routerId)
		fmt.Printf("最後のhello受信時刻: %v\n", neighborInfo.lastHelloTime)
		fmt.Printf("隣接ルーターへのリンク：リンク %v <-> %v\n", neighborInfo.link.NodeX().NodeId(), neighborInfo.link.NodeY().NodeId())
		fmt.Printf("追加情報 neighbors：")
		fmt.Println(neighborInfo.info.Neighbors)
	}
}

func (r *router) printTopologyDatabase() {
	fmt.Printf("========== TOPOLOGY DATABASE ==========（ルーター:%v）\n", r.NodeId())
	for routerID, entry := range r.topologyDatabase {
		fmt.Printf("Router ID: %d (Seq: %d)\n", routerID, entry.sequenceNumber)

		if len(entry.linkStateInfos) == 0 {
			fmt.Println("  [No Link State Information]")
			continue
		}

		// 各リンクの情報をループして出力
		for linkKey, info := range entry.linkStateInfos {
			fmt.Printf("  - Link [%s]: IP Address = %s, Cost = %.2f\n",
				linkKey, info.IpAddress, info.Cost)
		}
	}
	fmt.Println("=======================================")
}
