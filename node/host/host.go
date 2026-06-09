package host

import (
	"fmt"
	"math/rand/v2"
	"net"
	"nt-simulator/address"
	"nt-simulator/link"
	"nt-simulator/node/basenode"
	"nt-simulator/nteventsched"
	"nt-simulator/packet"
	"nt-simulator/packet/packetI"
	"sort"
	"strings"

	"github.com/google/uuid"
)

type dataWhenReceiveArpReply struct {
	data       string
	headerSize int
}

type dataWhenReceiveDNSReply struct {
	startTime   float64
	headerSize  int
	payloadSize int
}

// Nodeの構造体
type host struct {
	*basenode.BaseNode
	mtu               int
	fragmentedPackets map[string]map[int]packetI.PacketI
	*address.MacAddress
	*address.IpAddress
	arrivedCount       int
	receivedBytes      int
	arpTable           map[string]*address.MacAddress
	waitingForArpReply map[string][]*dataWhenReceiveArpReply
	waitingForDNSReply map[string][]*dataWhenReceiveDNSReply
	urlToIpMapping     map[string]string // urlとipアドレスのmapping
	dnsServerIp        string
}

func (n *host) ArrivedCount() int  { return n.arrivedCount }
func (n *host) ReceivedBytes() int { return n.receivedBytes }

func NewHost(nodeId int, ipAddress string, mtu int, nes *nteventsched.NtEventSched) *host {
	n := &host{BaseNode: basenode.NewBaseNode(nodeId, nes), fragmentedPackets: make(map[string]map[int]packetI.PacketI), mtu: mtu, MacAddress: address.NewMacAddress(address.GenerateRandomMAC()),
		IpAddress: address.NewIPAddress(ipAddress), arpTable: make(map[string]*address.MacAddress), waitingForArpReply: make(map[string][]*dataWhenReceiveArpReply), waitingForDNSReply: make(map[string][]*dataWhenReceiveDNSReply), urlToIpMapping: make(map[string]string), dnsServerIp: "192.168.1.200/24"}
	nes.AddNode(n)
	n.scheduleDHCPPacket()
	return n
}

func (n *host) PrintNode() {
	connected_nodes := make([]int, 0, 10)
	for _, v := range n.GetLinks() {
		if v.NodeX().NodeId() != n.NodeId() {
			connected_nodes = append(connected_nodes, v.NodeX().NodeId())
		}
		if v.NodeY().NodeId() != n.NodeId() {
			connected_nodes = append(connected_nodes, v.NodeY().NodeId())
		}
	}
	fmt.Printf("ノード(ID: %v, アドレス: %s), 接続ノード: %v\n", n.NodeId(), n.GetMacAddress(), connected_nodes)
}

func (n *host) NodeColor() string { return "" }

func (n *host) AddLink(link *link.Link, ip *address.IpAddress) {
	for _, l := range n.GetLinks() {
		if l == link {
			return
		}
	}
	n.SetLinks(append(n.GetLinks(), link))
}

func (n *host) ReceivePacket(p packetI.PacketI, l *link.Link) {
	if p.ArrivalTime() == -1 {
		n.GetNES().LogPacketInfo(p, "lost", n.NodeId())
		return
	}
	// arppacketを受け取った場合の処理
	// 自分が知りたいdest ipアドレスだったらreplyを返す
	if p.GetMacHeader().DestinationMac.String() == "FF:FF:FF:FF:FF:FF" {
		if arpP, ok := p.(*packet.ArpP); ok {
			ap, err := arpP.ParsePayload()
			if err != nil {
				fmt.Printf("arp parse error: %v\n", err)
				return
			}
			if ap.Operation == "request" && ap.DestIp == n.GetIPAddresses()[0].String() {
				n.sendArpReply(arpP)
				return
			}
		}
	}

	// 自分が知りたいIPアドレスだった場合の処理
	if p.GetMacHeader().DestinationMac.String() == n.MacAddress.String() && p.GetIpHeader().DestIp.String() == n.IpAddress.String() {
		// arpリプライが返ってきた場合の処理
		if arpP, ok := p.(*packet.ArpP); ok {
			ap, err := arpP.ParsePayload()
			if err != nil {
				fmt.Printf("arp parse error: %v\n", err)
				return
			}
			if ap.Operation == "reply" && ap.DestIp == n.GetIPAddresses()[0].String() {
				n.GetNES().LogPacketInfo(arpP, "ARP Reply received", n.NodeId())
				sourceIp := address.NewIPAddress(ap.SourceIp)
				sourceMac := address.NewMacAddress(ap.SourceMac)
				n.AddToArpTable(sourceIp, sourceMac)
				n.onArpReplyPacketReceived(ap.SourceIp)
				return
			}
		}

		if dnsP, ok := p.(*packet.DNSP); ok {
			dp, err := dnsP.ParsePayload()
			if err != nil {
				fmt.Printf("dns parse error: %v\n", err)
				return
			}
			n.GetNES().LogPacketInfo(dnsP, "DNS Reply received", n.NodeId())
			if dp.QueryDomain != "" && dp.ResolvedIp != "" {
				n.onDNSReplyPacketReceived(dp.QueryDomain, dp.ResolvedIp)
			}
			return
		}

		n.GetNES().LogPacketInfo(p, "arrived", n.NodeId())
		p.SetArrived(n.GetNES().CurrentTime)
		n.arrivedCount++
		n.receivedBytes += len(p.GetPayload())

		if p.GetIpHeader().FragmentFlags.MoreFragment {
			n.storeFragment(p)
		} else if p.GetIpHeader().FragmentOffset > 0 {
			n.reassembleAndProcessPacket(p)
		} else {
			// フラグメント化されていない単体パケット
			fmt.Printf("payload: %s\n", p.GetPayload())
			n.GetNES().LogPacketInfo(p, "processed", n.NodeId())
		}
	} else {
		n.GetNES().LogPacketInfo(p, "received", n.NodeId())
		// 宛先が自分自身にない場合
	}
}

func (n *host) GetIPAddresses() []*address.IpAddress {
	return []*address.IpAddress{n.IpAddress}
}

func (n *host) PrintArpTable() {
	fmt.Printf("--- ARP Table (%v) ---\n", n.NodeId()) // もしホスト名などがあれば
	fmt.Printf("%-15s   %-17s\n", "IP Address", "MAC Address")
	fmt.Println("---------------------------------------")

	if len(n.arpTable) == 0 {
		fmt.Println("(No entries found)")
		return
	}

	for ip, mac := range n.arpTable {
		// 左詰めで綺麗に並べて表示
		fmt.Printf("%-15s   %-17s\n", ip, mac)
	}
	fmt.Println("---------------------------------------")
}

func (n *host) StartTraffic(destinationURL string, startTime float64, headerSize int, payloadSize int) {
	n.GetNES().ScheduleEvent(startTime, func(args ...any) {
		n.attemptToStartTraffic(destinationURL, startTime, headerSize, payloadSize)
	})
}

func (n *host) scheduleDHCPPacket() {
	if n.IpAddress.IsNetworkAddress() {
		randomDelay := 0.5 + rand.Float64()*0.1
		n.GetNES().ScheduleEvent(n.GetNES().CurrentTime+randomDelay, func(args ...any) { n.sendDHCPDiscover() })
	}
}

func (n *host) sendDHCPDiscover() {

}

func (n *host) setTraffic(destinationIp *address.IpAddress, startTime float64, headerSize int, payloadSize int) {
	// endTime := startTime + duration
	// packetSize := headerSize + payloadSize
	// burstinessはよくわからん
	// このintervalで送れば，理論上指定したbitrateになる．
	// interval := float64(packetSize*8) / bitrate * burstiness

	// 全部のcreatePacketのスケジュールを最初にしておく
	// for t := startTime; t < endTime; t += interval {
	// 	n.GetNES().ScheduleEvent(t, func(args ...any) {
	// 		n.createPacket(address.NewMacAddress(destinationMac), address.NewIPAddress(destinationIp), headerSize, payloadSize, strings.Repeat("X", payloadSize))
	// 	})
	// }
	sendTime := startTime
	if n.GetNES().CurrentTime > sendTime {
		sendTime = n.GetNES().CurrentTime
	}
	n.GetNES().ScheduleEvent(sendTime, func(args ...any) {
		n.createPacket(destinationIp, headerSize, payloadSize, strings.Repeat("X", payloadSize))
	})
}

func (n *host) attemptToStartTraffic(destinationURL string, startTime float64, headerSize int, payloadSize int) {
	destinationIP, ok := n.resolveDestinationIp(destinationURL)
	if !ok {
		n.sendDNSQueryAndSetTraffic(destinationURL, startTime, headerSize, payloadSize)
	} else {
		n.setTraffic(address.NewIPAddress(destinationIP), startTime, headerSize, payloadSize)
	}
}

func (n *host) sendDNSQueryAndSetTraffic(destinationURL string, startTime float64, headerSize int, payloadSize int) {
	n.waitingForDNSReply[destinationURL] = append(n.waitingForDNSReply[destinationURL], &dataWhenReceiveDNSReply{
		startTime:   startTime,
		headerSize:  headerSize,
		payloadSize: payloadSize,
	})
	n.sendDNSQuery(destinationURL)
}

func (n *host) sendDNSQuery(destinationURL string) {
	p := packet.NewDNSP(n.MacAddress, address.NewMacAddress("FF:FF:FF:FF:FF:FF"), n.IpAddress, address.NewIPAddress(n.dnsServerIp), n.GetNES().CurrentTime, destinationURL, "A", "")
	n.GetNES().LogPacketInfo(p, "DNS Query", n.NodeId())
	n.internalSendPacket(p)
}

// fragmentedPacketsにoriginalDataIdのところにoffset付きで保管する
func (n *host) storeFragment(fragment packetI.PacketI) {
	originalDataId := fragment.GetIpHeader().FragmentFlags.OriginalDataId
	offset := fragment.GetIpHeader().FragmentOffset

	if _, ok := n.fragmentedPackets[originalDataId]; !ok {
		n.fragmentedPackets[originalDataId] = make(map[int]packetI.PacketI)
	}

	n.fragmentedPackets[originalDataId][offset] = fragment
	n.GetNES().LogPacketInfo(fragment, fmt.Sprintf("fragment_stored offset:%v originalDataId:%s moreflagment:%v", fragment.GetIpHeader().FragmentOffset, fragment.GetIpHeader().FragmentFlags.OriginalDataId, fragment.GetIpHeader().FragmentFlags.MoreFragment), n.NodeId())
}

func (n *host) reassembleAndProcessPacket(lastFragment packetI.PacketI) {
	originalDataId := lastFragment.GetIpHeader().FragmentFlags.OriginalDataId
	if _, ok := n.fragmentedPackets[originalDataId]; !ok {
		// 対応するフラグメントがない場合
		n.GetNES().LogPacketInfo(lastFragment, "reassemble failed no fragments", n.NodeId())
	}

	fragment_maps := n.fragmentedPackets[originalDataId]
	offsets := make([]int, 0, len(fragment_maps))
	for k := range fragment_maps {
		offsets = append(offsets, k)
	}
	sort.Ints(offsets)

	// 再組立されたデータをFragmentOffset順に結合して再構築
	var assembledPayload string
	for _, offset := range offsets {
		assembledPayload += fragment_maps[offset].GetPayload()
	}

	assembledPayload += lastFragment.GetPayload()

	expectedLength := lastFragment.GetIpHeader().FragmentOffset + len(lastFragment.GetPayload())
	if len(assembledPayload) != expectedLength {
		n.GetNES().LogPacketInfo(lastFragment, fmt.Sprintf("reassemble failed: missing fragments (expected %d bytes, got %d bytes)", expectedLength, len(assembledPayload)), n.NodeId())
		return
	}
	n.GetNES().LogPacketInfo(lastFragment, "reassembled", n.NodeId())
}

func (n *host) internalSendPacket(p packetI.PacketI) {
	n.GetNES().LogPacketInfo(p, "sent", n.NodeId())
	if p.GetMacHeader().DestinationMac == n.MacAddress {
		n.ReceivePacket(p, nil)
	} else {
		for _, l := range n.GetLinks() {
			var from_node *host = n
			l.EnqueuePacket(p, from_node)
		}
	}
}

func (n *host) sendPacket(destinationIp *address.IpAddress, data string, headerSize int) {
	payloadSize := n.mtu - headerSize
	totalSize := len(data) // goだとこれはバイト数になる
	offset := 0
	destinationMac := n.getMacAddressFromIp(destinationIp) // destinationIPアドレスからmacアドレスをひく

	// 宛先IPアドレスに対応するMacアドレスが未知の場合、arpリクエストを送信して終わり
	if destinationMac == nil {
		// ARPリクエストを送信して、パケットを待機リストに追加する
		n.sendArpRequest(destinationIp)
		n.waitingForArpReply[destinationIp.String()] = append(n.waitingForArpReply[destinationIp.String()], &dataWhenReceiveArpReply{data: data, headerSize: headerSize})
		return
	}

	originalDataId := uuid.New().String()

	for offset < totalSize {
		moreFragment := (offset + payloadSize) < totalSize

		end := offset + payloadSize
		if end > totalSize {
			end = totalSize
		}
		fragmentData := data[offset:end]
		fragmentOffset := offset

		packet := packet.NewFragment(n.MacAddress, destinationMac, n.IpAddress, destinationIp, 64, headerSize, n.GetNES().CurrentTime, originalDataId, moreFragment, fragmentOffset, fragmentData)

		n.internalSendPacket(packet) // 細かくfragmentにしたpacketを送信する
		offset += payloadSize
	}
}

func (n *host) sendArpRequest(ipAddress *address.IpAddress) {
	// ブロードキャスト
	arpPacket := packet.NewArpP(n.MacAddress, address.NewMacAddress("FF:FF:FF:FF:FF:FF"), n.IpAddress, ipAddress, n.GetNES().CurrentTime, "request")
	n.GetNES().LogPacketInfo(arpPacket, "ARP request", n.NodeId())
	n.internalSendPacket(arpPacket)
}

func (n *host) createPacket(destinationIp *address.IpAddress, headerSize int, payloadSize int, payload string) {
	destinationMac := n.getMacAddressFromIp(destinationIp)
	p := packet.NewPacket(n.MacAddress, destinationMac, n.IpAddress, destinationIp, 64, headerSize, payloadSize, n.GetNES().CurrentTime, payload)
	n.GetNES().LogPacketInfo(p, "created", n.NodeId())
	n.sendPacket(destinationIp, payload, headerSize)
}

func (n *host) AddToArpTable(ipAddress *address.IpAddress, macAddress *address.MacAddress) {
	n.arpTable[ipAddress.String()] = macAddress
}

func (n *host) getMacAddressFromIp(ipAddress *address.IpAddress) *address.MacAddress {
	macAddress, ok := n.arpTable[ipAddress.String()]
	if ok {
		return macAddress
	} else {
		return nil
	}
}

func (n *host) printArpTable() {
	fmt.Printf("--- ARP Table (%s) ---\n", n.NodeId()) // もしホスト名などがあれば
	fmt.Printf("%-15s   %-17s\n", "IP Address", "MAC Address")
	fmt.Println("---------------------------------------")

	if len(n.arpTable) == 0 {
		fmt.Println("(No entries found)")
		return
	}

	for ipStr, mac := range n.arpTable {
		macStr := "<nil>"
		if mac != nil {
			macStr = mac.String()
		}
		fmt.Printf("%-15s   %-17s\n", ipStr, macStr)
	}
	fmt.Println("---------------------------------------")
}

// DNSリプライを受信したら、待機中のパケットに対して処理を行う
func (n *host) onDNSReplyPacketReceived(query string, ipAddress string) {
	n.addDNSRecord(query, ipAddress)
	if _, ok := n.waitingForDNSReply[query]; ok {
		destinationIP := address.NewIPAddress(ipAddress)
		for _, v := range n.waitingForDNSReply[query] {
			n.setTraffic(destinationIP, v.startTime, v.headerSize, v.payloadSize)
		}
		n.waitingForDNSReply[query] = []*dataWhenReceiveDNSReply{}
	}
}

func (n *host) addDNSRecord(queryDomain string, resolvedIp string) {
	n.urlToIpMapping[queryDomain] = resolvedIp
}

// arpリプライを受信したら、待機中のパケットに対して処理を行う
func (n *host) onArpReplyPacketReceived(ipAddress string) {
	if _, ok := n.waitingForArpReply[ipAddress]; ok {
		destinationIP := address.NewIPAddress(ipAddress)
		for _, v := range n.waitingForArpReply[ipAddress] {
			n.sendPacket(destinationIP, v.data, v.headerSize)
		}
		n.waitingForArpReply[ipAddress] = []*dataWhenReceiveArpReply{}
	}
}

// arpリクエストを受け取って、宛先IPがこのノードのIPと一致していた場合の処理
func (n *host) sendArpReply(rp packetI.PacketI) {
	// 送られてきた元のノードに送り返す
	arpReplyPacket := packet.NewArpP(n.GetMacAddress(), rp.GetMacHeader().SourceMac, n.GetIPAddresses()[0], rp.GetIpHeader().SourceIp, n.GetNES().CurrentTime, "reply")
	n.GetNES().LogPacketInfo(arpReplyPacket, "ARP Reply", n.NodeId())
	n.internalSendPacket(arpReplyPacket)
}

func (n *host) resolveDestinationIp(destionaionUrl string) (string, bool) {
	if v, ok := n.urlToIpMapping[destionaionUrl]; ok {
		return v, true
	}
	if _, _, err := net.ParseCIDR(destionaionUrl); err == nil {
		return destionaionUrl, true
	}
	return "", false
}
