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
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
)

type dataWhenReceiveArpReply struct {
	data            string
	sourcePort      int
	destinationPort int
	protocol        string
}

type dataWhenReceiveDNSReply struct {
	startTime   float64
	headerSize  int
	payloadSize int
	protocol    string
}

type TCPConnectionKey struct {
	destinationIP   string
	destinationPort int
}

type pendingTCPData struct {
	destinationIp   string
	data            string
	sourcePort      int
	destinationPort int
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
	tcpConnections     map[TCPConnectionKey]packet.TCPConnectionState
	pendingTCPData     map[TCPConnectionKey]pendingTCPData
}

func (n *host) ArrivedCount() int  { return n.arrivedCount }
func (n *host) ReceivedBytes() int { return n.receivedBytes }

func NewHost(nodeId int, ipAddress string, mtu int, nes *nteventsched.NtEventSched) *host {
	n := &host{BaseNode: basenode.NewBaseNode(nodeId, nes), fragmentedPackets: make(map[string]map[int]packetI.PacketI), mtu: mtu, MacAddress: address.NewMacAddress(address.GenerateRandomMAC()),
		IpAddress: address.NewIPAddress(ipAddress), arpTable: make(map[string]*address.MacAddress), waitingForArpReply: make(map[string][]*dataWhenReceiveArpReply), waitingForDNSReply: make(map[string][]*dataWhenReceiveDNSReply), urlToIpMapping: make(map[string]string), dnsServerIp: "192.168.1.200/24", tcpConnections: make(map[TCPConnectionKey]packet.TCPConnectionState), pendingTCPData: make(map[TCPConnectionKey]pendingTCPData)}
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
	} else if arpP, ok := p.(*packet.ArpP); ok {
		n.processARPPacket(arpP)
	} else if dhcpP, ok := p.(*packet.DHCPP); ok {
		n.processDHCPPacket(dhcpP)
	} else if dnsP, ok := p.(*packet.DNSP); ok {
		n.processDNSPacket(dnsP)
	} else if udpP, ok := p.(*packet.UDPP); ok { // UDPパケットの処理
		n.processUDPPacket(udpP)
	} else if tcpP, ok := p.(*packet.TCPP); ok { // TCPパケットの処理
		n.processTCPPacket(tcpP)
	} else {
		n.GetNES().LogPacketInfo(p, "dropped", n.NodeId())
	}
}

func (n *host) processARPPacket(arpP *packet.ArpP) {
	if arpP.GetMacHeader().DestinationMac.String() == address.BroadcastMacAddress.String() {
		ap, err := arpP.ParsePayload()
		if err != nil {
			fmt.Printf("arp parse error: %v\n", err)
			return
		}
		if ap.Operation == packet.ArpOperationRequest && ap.DestIp == n.GetIPAddresses()[0].String() {
			n.sendArpReply(arpP)
			return
		}
	}

	if arpP.GetMacHeader().DestinationMac.String() == n.MacAddress.String() && arpP.GetIpHeader().DestIp.String() == n.IpAddress.String() {
		ap, err := arpP.ParsePayload()
		if err != nil {
			fmt.Printf("arp parse error: %v\n", err)
			return
		}
		if ap.Operation == packet.ArpOperationReply && ap.DestIp == n.GetIPAddresses()[0].String() {
			n.GetNES().LogPacketInfo(arpP, "ARP Reply received", n.NodeId())
			sourceIp := address.NewIPAddress(ap.SourceIp)
			sourceMac := address.NewMacAddress(ap.SourceMac)
			n.AddToArpTable(sourceIp, sourceMac)
			n.onArpReplyPacketReceived(ap.SourceIp)
			return
		}
	}
}

func (n *host) processDHCPPacket(dhcpP *packet.DHCPP) {
	if dhcpP.GetMacHeader().DestinationMac.String() == n.MacAddress.String() {
		dhcpPayload, err := dhcpP.ParsePayload()
		if err != nil {
			fmt.Printf("dhcp parse error: %v\n", err)
			return
		}
		switch dhcpPayload.MessageType {
		case packet.DHCPMessageTypeOffer:
			n.GetNES().LogPacketInfo(dhcpP, fmt.Sprintf("DHCP offer received: %s", dhcpPayload.OfferedIP), n.NodeId())
			n.sendDHCPRequest(dhcpPayload.OfferedIP)
			return
		case packet.DHCPMessageTypeACK:
			n.GetNES().LogPacketInfo(dhcpP, fmt.Sprintf("DHCP ack received: %s", dhcpPayload.AssignedIP), n.NodeId())
			n.IpAddress = address.NewIPAddress(dhcpPayload.AssignedIP)
			n.dnsServerIp = dhcpPayload.DnsServerIP
			return
		}
	}
}

func (n *host) processDNSPacket(dnsP *packet.DNSP) {
	if dnsP.GetMacHeader().DestinationMac.String() == n.MacAddress.String() {
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
}

func (n *host) processUDPPacket(p *packet.UDPP) {
	if p.GetMacHeader().DestinationMac.String() == n.MacAddress.String() && p.GetIpHeader().DestIp.String() == n.IpAddress.String() {
		n.processDataPacket(p)
	}
}

// TCPパケットを受け取ったときの処理
func (n *host) processTCPPacket(p *packet.TCPP) {
	if p.GetMacHeader().DestinationMac.String() == n.MacAddress.String() {
		if p.GetIpHeader().DestIp.String() == n.IpAddress.String() {
			flags := strings.Split(p.TCPHeader.Flags, ",")
			if slices.Contains(flags, "SYN") && !slices.Contains(flags, "ACK") {
				// SYNパケットを受信した場合、SYN-ACKを送信
				n.sendTCPSYNACK(p)
				n.logTCPControlPacketProcessed(p)
			} else if slices.Contains(flags, "SYN") && slices.Contains(flags, "ACK") {
				// SYN-ACKパケットを受信した場合、ACKを送信
				n.sendTCPACK(p)
				n.logTCPControlPacketProcessed(p)
			} else if slices.Contains(flags, "ACK") {
				// ACKパケットを受信した場合、接続が確立されたとみなす
				n.establishTCPConnection(p)
				n.logTCPControlPacketProcessed(p)
			} else if slices.Contains(flags, "FIN") {
				// FINパケットを受信した場合、接続を修了
				n.terminateTCPConnection(p)
			} else {
				n.processDataPacket(p)
			}
		}
	}
}

func (n *host) logTCPControlPacketProcessed(p *packet.TCPP) {
	n.GetNES().LogPacketInfo(p, "arrived", n.NodeId())
	p.SetArrived(n.GetNES().CurrentTime)
	n.GetNES().LogPacketInfo(p, "processed", n.NodeId())
}

func (n *host) sendTCPSYNACK(p *packet.TCPP) {
	synAckPacketFlags := "SYN,ACK"
	n.sendTCPPacket(p.GetIpHeader().SourceIp, p.GetMacHeader().SourceMac, "", p.TCPHeader.DestinationPort, p.TCPHeader.SourcePort, synAckPacketFlags)
}

func (n *host) sendTCPACK(p *packet.TCPP) {
	ackPacketFlags := "ACK"
	n.sendTCPPacket(p.GetIpHeader().SourceIp, p.GetMacHeader().SourceMac, "", p.TCPHeader.DestinationPort, p.TCPHeader.SourcePort, ackPacketFlags)
	n.establishTCPConnection(p)
}

func (n *host) processDataPacket(p packetI.PacketI) {
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

func (n *host) StartTraffic(destinationURL string, startTime float64, headerSize int, payloadSize int, protocol string) {
	n.GetNES().ScheduleEvent(startTime, func(args ...any) {
		n.attemptToStartTraffic(destinationURL, startTime, headerSize, payloadSize, protocol)
	})
}

func (n *host) scheduleDHCPPacket() {
	if n.IpAddress.IsNetworkAddress() {
		randomDelay := 0.5 + rand.Float64()*0.1
		n.GetNES().ScheduleEvent(n.GetNES().CurrentTime+randomDelay, func(args ...any) { n.sendDHCPDiscover() })
	}
}

func (n *host) sendDHCPDiscover() {
	dhcpDiscoverPacket := packet.NewDHCPP(n.MacAddress, address.BroadcastMacAddress, address.DHCPSourceIPAddress, address.BroadcastIPAddress, n.GetNES().CurrentTime, packet.DHCPMessageTypeDiscover)
	n.internalSendPacket(dhcpDiscoverPacket)
}

func (n *host) sendDHCPRequest(requestedIP string) {
	dhcpRequestPacket := packet.NewDHCPPWithRequestedIP(n.MacAddress, address.BroadcastMacAddress, address.DHCPSourceIPAddress, address.BroadcastIPAddress, n.GetNES().CurrentTime, packet.DHCPMessageTypeRequest, requestedIP)
	n.internalSendPacket(dhcpRequestPacket)
}

func (n *host) setTraffic(destinationIp *address.IpAddress, startTime float64, headerSize int, payloadSize int, protocol string) {
	sendTime := startTime
	sourcePort := n.selectRandomPort()
	destinationPort := n.selectRandomPort()
	if n.GetNES().CurrentTime > sendTime {
		sendTime = n.GetNES().CurrentTime
	}
	n.GetNES().ScheduleEvent(sendTime, func(args ...any) {
		payload := strings.Repeat("X", payloadSize)
		destinationMac := n.getMacAddressFromIp(destinationIp)
		p := packet.NewPacket(n.MacAddress, destinationMac, n.IpAddress, destinationIp, 64, headerSize, payloadSize, n.GetNES().CurrentTime, payload)
		n.GetNES().LogPacketInfo(p, "created", n.NodeId())
		switch protocol {
		case "UDP":
			n.sendUDPPacket(destinationIp, payload, sourcePort, destinationPort)
		case "TCP":
			n.startTCPConnectionAndSendPacket(destinationIp, payload, sourcePort, destinationPort, "")
		}
	})
}

func (n *host) attemptToStartTraffic(destinationURL string, startTime float64, headerSize int, payloadSize int, protocol string) {
	destinationIP, ok := n.resolveDestinationIp(destinationURL)
	if !ok {
		n.sendDNSQueryAndSetTraffic(destinationURL, startTime, headerSize, payloadSize, protocol)
	} else {
		n.setTraffic(address.NewIPAddress(destinationIP), startTime, headerSize, payloadSize, protocol)
	}
}

func (n *host) sendDNSQueryAndSetTraffic(destinationURL string, startTime float64, headerSize int, payloadSize int, protocol string) {
	n.waitingForDNSReply[destinationURL] = append(n.waitingForDNSReply[destinationURL], &dataWhenReceiveDNSReply{
		startTime:   startTime,
		headerSize:  headerSize,
		payloadSize: payloadSize,
		protocol:    protocol,
	})
	p := packet.NewDNSP(n.MacAddress, address.BroadcastMacAddress, n.IpAddress, address.NewIPAddress(n.dnsServerIp), n.GetNES().CurrentTime, destinationURL, packet.DNSQueryTypeA, "")
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

func (n *host) sendUDPPacket(destinationIp *address.IpAddress, data string, sourcePort int, destinationPort int) {
	destinationMac := n.getMacAddressFromIp(destinationIp) // destinationIPアドレスからmacアドレスをひく

	// 宛先IPアドレスに対応するMacアドレスが未知の場合、arpリクエストを送信して終わり
	if destinationMac == nil {
		// ARPリクエストを送信して、パケットを待機リストに追加する
		n.sendArpRequest(destinationIp)
		n.waitingForArpReply[destinationIp.String()] = append(n.waitingForArpReply[destinationIp.String()], &dataWhenReceiveArpReply{data: data, sourcePort: sourcePort, destinationPort: destinationPort, protocol: "UDP"})
		return
	}
	udpHeaderSize := 8
	ipHeaderSize := 20
	headerSize := udpHeaderSize + ipHeaderSize
	payloadSize := n.mtu - headerSize
	totalSize := len(data) // goだとこれはバイト数になる
	offset := 0
	originalDataId := uuid.New().String()

	for offset < totalSize {
		moreFragment := (offset + payloadSize) < totalSize

		end := offset + payloadSize
		if end > totalSize {
			end = totalSize
		}
		fragmentData := data[offset:end]
		fragmentOffset := offset
		p := packet.NewUDPPacket(n.MacAddress, destinationMac, n.IpAddress, destinationIp, 64, headerSize, n.GetNES().CurrentTime, originalDataId, moreFragment, fragmentOffset, fragmentData, sourcePort, destinationPort)

		n.internalSendPacket(p) // 細かくfragmentにしたpacketを送信する
		offset += payloadSize
	}
}

func (n *host) startTCPConnectionAndSendPacket(destinationIp *address.IpAddress, data string, sourcePort int, destinationPort int, flags string) {
	destinationMac := n.getMacAddressFromIp(destinationIp) // destinationIPアドレスからmacアドレスをひく

	// 宛先IPアドレスに対応するMacアドレスが未知の場合、arpリクエストを送信して終わり
	if destinationMac == nil {
		// ARPリクエストを送信して、パケットを待機リストに追加する
		n.sendArpRequest(destinationIp)
		n.waitingForArpReply[destinationIp.String()] = append(n.waitingForArpReply[destinationIp.String()], &dataWhenReceiveArpReply{data: data, sourcePort: sourcePort, destinationPort: destinationPort, protocol: "TCP"})
		return
	}
	if !n.isTCPConnectionEstablished(destinationIp, destinationPort) {
		connectionKey := TCPConnectionKey{destinationIP: destinationIp.String(), destinationPort: destinationPort}
		n.pendingTCPData[connectionKey] = pendingTCPData{destinationIp: destinationIp.String(), destinationPort: destinationPort, sourcePort: sourcePort, data: data}
		// 接続が未確立なので、ハンドシェイクを開始。これは実際にはaccept関数がやってくれる
		n.initiateTCPHandshake(destinationIp, destinationMac, sourcePort, destinationPort)
	}else{
	n.sendTCPPacket(destinationIp, destinationMac, data, sourcePort, destinationPort, flags)
	}
}

func (n *host) sendTCPPacket(destinationIp *address.IpAddress, destinationMac *address.MacAddress, data string, sourcePort int, destinationPort int, flags string) {
	tcpHeaderSize := 20
	ipHeaderSize := 20
	headerSize := tcpHeaderSize + ipHeaderSize
	payloadSize := n.mtu - headerSize
	totalSize := len(data) // goだとこれはバイト数になる
	offset := 0
	originalDataId := uuid.New().String()

	// SYN/ACK などペイロードなしの制御パケットも1回は送信する
	if totalSize == 0 {
		p := packet.NewTCPPacket(n.MacAddress, destinationMac, n.IpAddress, destinationIp, 64, headerSize, n.GetNES().CurrentTime, originalDataId, false, 0, "", sourcePort, destinationPort, 0, 0, flags)
		n.internalSendPacket(p)
		return
	}

	for offset < totalSize {
		moreFragment := (offset + payloadSize) < totalSize

		end := offset + payloadSize
		if end > totalSize {
			end = totalSize
		}
		fragmentData := data[offset:end]
		fragmentOffset := offset
		p := packet.NewTCPPacket(n.MacAddress, destinationMac, n.IpAddress, destinationIp, 64, headerSize, n.GetNES().CurrentTime, originalDataId, moreFragment, fragmentOffset, fragmentData, sourcePort, destinationPort, 0, 0, flags)

		n.internalSendPacket(p) // 細かくfragmentにしたpacketを送信する
		offset += payloadSize
	}
}

func (n *host) sendArpRequest(ipAddress *address.IpAddress) {
	// ブロードキャスト
	arpPacket := packet.NewArpP(n.MacAddress, address.BroadcastMacAddress, n.IpAddress, ipAddress, n.GetNES().CurrentTime, packet.ArpOperationRequest)
	n.GetNES().LogPacketInfo(arpPacket, "ARP request", n.NodeId())
	n.internalSendPacket(arpPacket)
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
			n.setTraffic(destinationIP, v.startTime, v.headerSize, v.payloadSize, v.protocol)
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
			switch v.protocol {
			case "UDP":
				n.sendUDPPacket(destinationIP, v.data, v.sourcePort, v.destinationPort)
			case "TCP":
				n.startTCPConnectionAndSendPacket(destinationIP, v.data, v.sourcePort, v.destinationPort, "")
			}
		}
		n.waitingForArpReply[ipAddress] = []*dataWhenReceiveArpReply{}
	}
}

// arpリクエストを受け取って、宛先IPがこのノードのIPと一致していた場合の処理
func (n *host) sendArpReply(rp packetI.PacketI) {
	// 送られてきた元のノードに送り返す
	arpReplyPacket := packet.NewArpP(n.GetMacAddress(), rp.GetMacHeader().SourceMac, n.GetIPAddresses()[0], rp.GetIpHeader().SourceIp, n.GetNES().CurrentTime, packet.ArpOperationReply)
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

func (n *host) selectRandomPort() int {
	return rand.N(49151-1024+1) + 1024
}

func (n *host) isTCPConnectionEstablished(destinationIP *address.IpAddress, destinationPort int) bool {
	if status, ok := n.tcpConnections[TCPConnectionKey{destinationIP: destinationIP.String(), destinationPort: destinationPort}]; ok {
		if status == packet.TCPConnectionStateEstablished {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

// TCP接続を確立する。接続が確立されたら、保存しておいたデータを送信する
func (n *host) establishTCPConnection(p *packet.TCPP) {
	tcpConnectionKey := TCPConnectionKey{destinationIP: p.GetIpHeader().SourceIp.String(), destinationPort: p.TCPHeader.SourcePort}
	if n.isTCPConnectionEstablished(p.GetIpHeader().SourceIp, p.TCPHeader.SourcePort) {
		return
	}

	n.updateTCPConnectionState(p.GetIpHeader().SourceIp.String(), p.TCPHeader.SourcePort, packet.TCPConnectionStateEstablished)
	// 保存しておいたデータがあれば送信する
	if pendingData, ok := n.pendingTCPData[tcpConnectionKey]; ok {
		delete(n.pendingTCPData, tcpConnectionKey)
		n.sendTCPPacket(p.GetIpHeader().SourceIp, p.GetMacHeader().SourceMac, pendingData.data, pendingData.sourcePort, pendingData.destinationPort, "")
	}
}

func (n *host) initiateTCPHandshake(destinationIp *address.IpAddress, destinationMac *address.MacAddress, sourcePort int, destinationPort int) {
	// establishedじゃなかったらSYNパケットを送信
	if !n.isTCPConnectionEstablished(destinationIp, destinationPort) {
		// 接続状態をSYN SENTに更新
		n.updateTCPConnectionState(destinationIp.String(), destinationPort, packet.TCPConnectionStateSynSent)
		// flagsを"SYN"にして送信
		n.sendTCPPacket(destinationIp, destinationMac, "", sourcePort, destinationPort, "SYN")
	}
}

// 指定された宛先に対するTCP接続の状態を更新
func (n *host) updateTCPConnectionState(destinationIP string, destinatioPort int, newState packet.TCPConnectionState) {
	n.tcpConnections[TCPConnectionKey{destinationIP: destinationIP, destinationPort: destinatioPort}] = newState
}

func (n *host) terminateTCPConnection(p *packet.TCPP) {
	tcpConnectionKey := TCPConnectionKey{destinationIP: p.GetIpHeader().SourceIp.String(), destinationPort: p.TCPHeader.SourcePort}
	delete(n.tcpConnections, tcpConnectionKey)
}
