package router

import (
	"fmt"
	"nt-simulator/address"
	"nt-simulator/link"
	"nt-simulator/node/basenode"
	"nt-simulator/nteventsched"
	"nt-simulator/packet"
	"nt-simulator/packet/packetI"
)

/*
[ルーターA]----リンク----[ルーターB]
 10.0.0.1                10.0.0.2

ルーターA：
interfaces → このリンクに対して 10.0.0.1（自分のアドレス）
nexthop → 10.0.0.2（ルーターBのアドレス）
*/

type routingTableEntry struct {
	nexthop int
	link    *link.Link
}

type neighborInfo struct {
	lastHelloTime float64
	link          *link.Link
	info          packet.HellopPayload
}

type topologyEntry struct {
	sequenceNumber int
	linkStateInfos []packet.LinkStateInfo // そのrouterIDが繋がっているlinkのstate情報
}

type router struct {
	*basenode.BaseNode
	availableIps          map[*address.IpAddress]bool        // CIDR表記のIPアドレスを辞書に変換して，使用状況を記録
	interfaces            map[*link.Link]*address.IpAddress  // リンクとIPアドレスのマッピング
	macAddresses          map[*link.Link]*address.MacAddress // インタフェースとmacアドレスの組み合わせ
	arpTable              map[string]*address.MacAddress
	routingTable          map[*address.IpAddress]routingTableEntry
	helloInterval         float64
	neighbors             map[int]*neighborInfo
	lsaSequenceNumber     int
	lsaInterval           float64
	isTopologyInitialized bool
	topologyDatabase      map[int]topologyEntry // 各router idのリンク情報を管理
	waitingForArpReply    map[string][]packetI.PacketI
	natEnabled            bool
	externalIP            *address.IpAddress
	natTable              map[string]string
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
		macAddresses:          make(map[*link.Link]*address.MacAddress),
		arpTable:              make(map[string]*address.MacAddress),
		routingTable:          make(map[*address.IpAddress]routingTableEntry),
		helloInterval:         10.0,
		neighbors:             make(map[int]*neighborInfo),
		lsaInterval:           10.0,
		isTopologyInitialized: false,
		topologyDatabase:      make(map[int]topologyEntry), // ルーターIDごとのLsaの情報を管理
		waitingForArpReply:    make(map[string][]packetI.PacketI),
	}
	nes.AddNode(r)
	r.scheduleHelloPacket()
	r.scheduleLsaPacket()
	return r
}

func NewRouterNATEnabled(nodeId int, ipAddreses []string, nes *nteventsched.NtEventSched, externalIP *address.IpAddress) *router {
	availableIps := make(map[*address.IpAddress]bool)
	for _, ip := range ipAddreses {
		availableIps[address.NewIPAddress(ip)] = false
	}
	r := &router{
		BaseNode:              basenode.NewBaseNode(nodeId, nes),
		availableIps:          availableIps,
		interfaces:            make(map[*link.Link]*address.IpAddress),
		macAddresses:          make(map[*link.Link]*address.MacAddress),
		arpTable:              make(map[string]*address.MacAddress),
		routingTable:          make(map[*address.IpAddress]routingTableEntry),
		helloInterval:         10.0,
		neighbors:             make(map[int]*neighborInfo),
		lsaInterval:           10.0,
		isTopologyInitialized: false,
		topologyDatabase:      make(map[int]topologyEntry), // ルーターIDごとのLsaの情報を管理
		waitingForArpReply:    make(map[string][]packetI.PacketI),
		natEnabled:            true,
		externalIP:            externalIP,
		natTable:              make(map[string]string),
	}
	nes.AddNode(r)
	r.scheduleHelloPacket()
	r.scheduleLsaPacket()
	return r
}

func (r *router) NodeColor() string { return "blue" }

// リーターに新しいリンクを追加する
func (r *router) AddLink(link *link.Link, ipAddress *address.IpAddress) {
	if _, ok := r.interfaces[link]; !ok {
		r.interfaces[link] = ipAddress
		r.macAddresses[link] = address.NewMacAddress(address.GenerateRandomMAC())
		r.markIpAsUsed(ipAddress)
		networkCIDR := ipAddress.ConvertToNetworkCIDR()
		r.routingTable[networkCIDR] = routingTableEntry{nexthop: -1, link: link}
	}
}

// ルーティングテーブルにルートを追加するメソッド
// このdestinationCIDRはネットワークアドレス
func (r *router) AddRoute(destinationCIDR string, nexthop int, link *link.Link) {
	r.routingTable[address.NewIPAddress(destinationCIDR)] = routingTableEntry{nexthop: nexthop, link: link}
}

func (r *router) ReceivePacket(p packetI.PacketI, receivedLink *link.Link) {
	if arpP, ok := p.(*packet.ArpP); ok {
		ap, err := arpP.ParsePayload()
		if err != nil {
			fmt.Printf("arp parse error: %v\n", err)
			return
		}
		switch ap.Operation {
		case packet.ArpOperationRequest:
			{
				r.onArpRequestPacketReceived(arpP, receivedLink)
			}
		case packet.ArpOperationReply:
			{
				r.onArpReplyPacketReceived(p.GetIpHeader().SourceIp, p.GetMacHeader().SourceMac)
			}
		}
	}

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

	if p.GetMacHeader().DestinationMac.String() == r.GetMacAddress(receivedLink).String() {
		r.GetNES().LogPacketInfo(p, "router received", r.NodeId())
		p.RemoveMacHeader() // Macアドレスの付け替え処理
		// 普通のパケットの処理
		destIp := p.GetIpHeader().DestIp
		for _, interfaceCIDR := range r.interfaces {
			if destIp.IsSameNetwork(interfaceCIDR) {
				if destIp.String() == interfaceCIDR.String() { // パケットがルーター宛
					if r.natEnabled && r.externalIP != nil && destIp.String() == r.externalIP.String() {
						r.applyNAT(p, "inbound")
						r.forwardPacket(p)
						return
					}
					r.GetNES().LogPacketInfo(p, "arrived to router", r.NodeId())
				} else { // ルーター宛ではないが，そのルーターと同じネットワークだった場合
					r.forwardPacket(p)
				}
				return
			}
		}
		r.forwardPacket(p)
	} else {
		r.GetNES().LogPacketInfo(p, "dropped due to unmatched MAC Address", r.NodeId())
	}
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

// GetRoutingEntries はルーティングテーブルを "ネットワークCIDR -> nexthop router ID" のマップで返す。
// nexthop が -1 の場合は直接接続されたネットワーク。
func (r *router) GetRoutingEntries() map[string]int {
	result := make(map[string]int, len(r.routingTable))
	for ip, entry := range r.routingTable {
		result[ip.String()] = entry.nexthop
	}
	return result
}

func (r *router) GetMacAddress(link *link.Link) *address.MacAddress {
	return r.macAddresses[link]
}

func (r *router) PrintArpTable() {
	fmt.Printf("--- ARP Table (%v) ---\n", r.NodeId()) // もしホスト名などがあれば
	fmt.Printf("%-15s   %-17s\n", "IP Address", "MAC Address")
	fmt.Println("---------------------------------------")

	if len(r.arpTable) == 0 {
		fmt.Println("(No entries found)")
		return
	}

	for ip, mac := range r.arpTable {
		// 左詰めで綺麗に並べて表示
		fmt.Printf("%-15s   %-17s\n", ip, mac)
	}
	fmt.Println("---------------------------------------")
}

func (r *router) AddToArpTable(ipAddress *address.IpAddress, macAddress *address.MacAddress) {
	r.arpTable[ipAddress.String()] = macAddress
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
func (r *router) getRoute(destionationIp *address.IpAddress) (routingTableEntry, bool) {
	for networkCIDR, next := range r.routingTable {
		if networkCIDR.IsSameNetwork(destionationIp) {
			return next, true
		}
	}
	return routingTableEntry{}, false // ルートが見つからなかった場合
}

func (r *router) forwardPacket(p packetI.PacketI) {
	destinationAddress := p.GetIpHeader().DestIp
	entry, ok := r.getRoute(destinationAddress) // このdestination Addressに行くなら，このlinkを辿ってこのIPアドレスに行け

	// 宛先がOSPFのマルチキャストアドレスの場合
	if destinationAddress.String() == address.OSPFAllSPFRoutersIP {
		for l := range r.interfaces { // 全部のinterfaceにパケットを送信
			r.proceedAndEnqueuePacket(p, l)
		}
	} else if ok { // ルーティング成功
		r.proceedAndEnqueuePacket(p, entry.link)
	} else {
		r.GetNES().LogPacketInfo(p, "router dropped", r.NodeId())
	}
	// TODO：デフォルトルートの実装
}

// 異なるネットワークセグメント間では、macアドレスを付け替える必要がある
func (r *router) proceedAndEnqueuePacket(p packetI.PacketI, l *link.Link) {
	if r.natEnabled {
		sourceInternal := r.isInternalIp(p.GetIpHeader().SourceIp)
		destInternal := r.isInternalIp(p.GetIpHeader().DestIp)
		if sourceInternal && !destInternal {
			// 内部ネットワークから外部ネットワークへのパケット
			r.applyNAT(p, "outbound")
		} else if !sourceInternal && r.externalIP != nil && p.GetIpHeader().DestIp.String() == r.externalIP.String() {
			r.applyNAT(p, "inbound")
		}
	}

	sourceMac := r.GetMacAddress(l)
	destMac := r.getMacAddressFromIp(p.GetIpHeader().DestIp)
	destIp := p.GetIpHeader().DestIp

	// 宛先MACアドレスが不明の場合、ARPリクエストを送信してパケットを待機リストに追加する
	if destMac == nil {
		r.sendArpRequest(l, destIp)
		r.waitingForArpReply[destIp.String()] = append(r.waitingForArpReply[destIp.String()], p)
		return
	}
	p.AddMacHeader(sourceMac, destMac)
	r.GetNES().LogPacketInfo(p, "router forwarded", r.NodeId())
	l.EnqueuePacket(p, r)
}

func (r *router) sendArpRequest(l *link.Link, ipAddress *address.IpAddress) {
	arpPacket := packet.NewArpP(r.GetMacAddress(l), address.BroadcastMacAddress, r.getIpAddress(l), ipAddress, r.GetNES().CurrentTime, packet.ArpOperationRequest)
	r.GetNES().LogPacketInfo(arpPacket, "ARP request", r.NodeId())
	l.EnqueuePacket(arpPacket, r)
}

// arpリプライを受信したら、待機中のパケットに対して処理を行う
func (r *router) onArpReplyPacketReceived(destinationIP *address.IpAddress, destinationMac *address.MacAddress) {
	r.AddToArpTable(destinationIP, destinationMac)
	if _, ok := r.waitingForArpReply[destinationIP.String()]; ok {
		for _, p := range r.waitingForArpReply[destinationIP.String()] {
			r.forwardPacket(p)
		}
		r.waitingForArpReply[destinationIP.String()] = []packetI.PacketI{}
	}
}

// requestを受け取ったら、とりあえず自分のルーターに送れって指示を出す
// 多分これだと複数ルーターが繋がってたときによくないけど、simulatorの都合上、1つのnodeで繋がっているのはルーター1個ってことになってるのかな
func (r *router) onArpRequestPacketReceived(rp *packet.ArpP, l *link.Link) {
	arpReplyPacket := packet.NewArpP(r.GetMacAddress(l), rp.GetMacHeader().SourceMac, rp.GetIpHeader().DestIp, rp.GetIpHeader().SourceIp, r.GetNES().CurrentTime, packet.ArpOperationReply)
	r.GetNES().LogPacketInfo(arpReplyPacket, "ARP Reply", r.NodeId())
	l.EnqueuePacket(arpReplyPacket, r)
}

func (r *router) getIpAddress(link *link.Link) *address.IpAddress {
	return r.interfaces[link]
}

func (r *router) getMacAddressFromIp(ipAddress *address.IpAddress) *address.MacAddress {
	macAddress, ok := r.arpTable[ipAddress.String()]
	if ok {
		return macAddress
	} else {
		return nil
	}
}

func (r *router) applyNAT(p packetI.PacketI, direction string) {
	switch direction {
	case "outbound":
		originalSrcIP := p.GetIpHeader().SourceIp
		r.natTable[originalSrcIP.String()] = r.externalIP.String()
		header := p.GetIpHeader()
		header.SourceIp = r.externalIP
		p.SetIpHeader(header)
	case "inbound":
		internalIP, ok := r.lookupInternalIPByExternal(p.GetIpHeader().DestIp.String())
		if !ok {
			return
		}
		header := p.GetIpHeader()
		header.DestIp = address.NewIPAddress(internalIP)
		p.SetIpHeader(header)
	}
}

func (r *router) lookupInternalIPByExternal(externalIP string) (string, bool) {
	for internalIP, mappedExternalIP := range r.natTable {
		if mappedExternalIP == externalIP {
			return internalIP, true
		}
	}
	return "", false
}

// IPアドレスが InternalNetworkIP に属しているかを判断する
func (r *router) isInternalIp(ipAddress *address.IpAddress) bool {
	return ipAddress.IsSameNetwork(address.InternalNetworkIPAddress)
}
