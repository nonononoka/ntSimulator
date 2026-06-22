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
)

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
	tcpConnections     map[tcpConnectionKey]*tcpConnectionStateValue
	pendingTCPData     map[tcpConnectionKey]*pendingTCPData
	windows            map[tcpConnectionKey]int // ウインドウ内のパケットのシーケンス番号
	windowSize         int
}

func (n *host) ArrivedCount() int  { return n.arrivedCount }
func (n *host) ReceivedBytes() int { return n.receivedBytes }

func NewHost(nodeId int, ipAddress string, mtu int, nes *nteventsched.NtEventSched) *host {
	n := &host{BaseNode: basenode.NewBaseNode(nodeId, nes), fragmentedPackets: make(map[string]map[int]packetI.PacketI), mtu: mtu, MacAddress: address.NewMacAddress(address.GenerateRandomMAC()),
		IpAddress: address.NewIPAddress(ipAddress), arpTable: make(map[string]*address.MacAddress), waitingForArpReply: make(map[string][]*dataWhenReceiveArpReply), waitingForDNSReply: make(map[string][]*dataWhenReceiveDNSReply), urlToIpMapping: make(map[string]string), dnsServerIp: "192.168.1.200/24", tcpConnections: make(map[tcpConnectionKey]*tcpConnectionStateValue), pendingTCPData: make(map[tcpConnectionKey]*pendingTCPData), windows: make(map[tcpConnectionKey]int), windowSize: 65535}
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

func (n *host) GetIPAddresses() []*address.IpAddress {
	return []*address.IpAddress{n.IpAddress}
}

func (n *host) StartTraffic(destinationURL string, startTime float64, headerSize int, payloadSize int, protocol string) {
	n.GetNES().ScheduleEvent(startTime, func(args ...any) {
		n.attemptToStartTraffic(destinationURL, startTime, headerSize, payloadSize, protocol)
	})
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

// 本書のset_tcp_trafficに相当するつもり
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
			n.startTCPConnectionAndSendPacket(destinationIp, payload, sourcePort, destinationPort, sendTime)
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
