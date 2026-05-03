package network

import (
	"fmt"
	"nt-simulator/nteventsched"
	"nt-simulator/packet"
	"strings"

	"github.com/google/uuid"
)

// Nodeの構造体
type terminalN struct {
	nodeId            int
	macAddress        string
	ipAddress         string
	links             []*Link
	mtu               int
	fragmentedPackets map[string]map[int]packet.PacketI
	nes               *nteventsched.NtEventSched
}

func (n *terminalN) Address() string {
	return n.macAddress
}

func (n *terminalN) PrintNode() {
	connected_nodes := make([]int, 0, 10)
	for _, v := range n.links {
		if v.nodeX.NodeId() != n.nodeId {
			connected_nodes = append(connected_nodes, v.nodeX.NodeId())
		}
		if v.nodeY.NodeId() != n.nodeId {
			connected_nodes = append(connected_nodes, v.nodeY.NodeId())
		}
	}
	fmt.Printf("ノード(ID: %v, アドレス: %s), 接続ノード: %v\n", n.nodeId, n.macAddress, connected_nodes)
}

func (n *terminalN) NodeId() int {
	return n.nodeId
}

func (n *terminalN) NodeColor() string { return "" }

func (n *terminalN) AddLink(link *Link) {
	for _, l := range n.links {
		if l == link {
			return
		}
	}
	n.links = append(n.links, link)
}

func NewNode(node_id int, macaddress string, ipaddress string, mtu int, nes *nteventsched.NtEventSched) (*terminalN, error) {
	if !isValidMacAddress(macaddress) {
		return nil, fmt.Errorf("invalid MAC address: %s", macaddress)
	}
	if !isValidIpV4Address(ipaddress) {
		return nil, fmt.Errorf("invalid ip address: %s", macaddress)
	}
	n := &terminalN{nodeId: node_id, macAddress: macaddress, ipAddress: ipaddress, fragmentedPackets: make(map[string]map[int]packet.PacketI), mtu: mtu, nes: nes}
	nes.AddNode(n)
	return n, nil
}

func (n *terminalN) receivePacket(p packet.PacketI, l *Link) {
	if p.ArrivalTime() == -1 {
		n.nes.LogPacketInfo(p, "lost", n.nodeId)
		return
	}
	if p.GetHeader().DestinationMac == n.macAddress && p.GetHeader().DestIp == n.ipAddress {
		n.nes.LogPacketInfo(p, "arrived", n.nodeId)
		p.SetArrived(n.nes.CurrentTime)

		if p.GetHeader().FragmentFlags.MoreFragment {
			n.storeFragment(p)
		} else if p.GetHeader().FragmentOffset > 0 {
			n.reassembleAndProcessPacket(p)
		} else {
			// フラグメント化されていない単体パケット
			fmt.Printf("payload: %s\n", p.GetPayload())
			n.nes.LogPacketInfo(p, "processed", n.nodeId)
		}
	} else {
		n.nes.LogPacketInfo(p, "received", n.nodeId)
		// 宛先が自分自身にない場合
	}
}

// fragmentedPacketsにoriginalDataIdのところにoffset付きで保管する
func (n *terminalN) storeFragment(fragment packet.PacketI) {
	originalDataId := fragment.GetHeader().FragmentFlags.OriginalDataId
	offset := fragment.GetHeader().FragmentOffset

	if _, ok := n.fragmentedPackets[originalDataId][offset]; !ok {
		n.fragmentedPackets[originalDataId] = make(map[int]packet.PacketI)
	}

	n.fragmentedPackets[originalDataId][offset] = fragment
	n.nes.LogPacketInfo(fragment, fmt.Sprintf("fragment_stored offset:%v originalDataId:%s moreflagment:%v", fragment.GetHeader().FragmentOffset, fragment.GetHeader().FragmentFlags.OriginalDataId, fragment.GetHeader().FragmentFlags.MoreFragment), n.nodeId)
}

func (n *terminalN) reassembleAndProcessPacket(lastFragment packet.PacketI) {
	originalDataId := lastFragment.GetHeader().FragmentFlags.OriginalDataId
	if _, ok := n.fragmentedPackets[originalDataId]; !ok {
		// 対応するフラグメントがない場合
		n.nes.LogPacketInfo(lastFragment, "reassemble failed no fragments", n.NodeId())
	}

	fragment_maps := n.fragmentedPackets[originalDataId]
	fragments := make([]packet.PacketI, 0, len(fragment_maps))
	for _, v := range fragment_maps {
		fragments = append(fragments, v)
	}

	// 再組立されたデータを結合して再構築
	var assembledPayload string
	for _, p := range fragments {
		assembledPayload += p.GetPayload()
	}

	assembledPayload += lastFragment.GetPayload()
	// fmt.Printf("assembled payload: %s\n", assembledPayload)
	// TODO 期待される長さと実際の総データ長を比較して，欠けているフラグメントをチェック
	n.nes.LogPacketInfo(lastFragment, "reassembled", n.nodeId)
}

func (n *terminalN) sendPacket(p *packet.Packet) {
	n.nes.LogPacketInfo(p, "sent", n.nodeId)
	if p.Header.DestinationMac == n.macAddress {
		n.receivePacket(p, nil)
	} else {
		for _, l := range n.links {
			var from_node *terminalN = n
			l.enqueuePacket(p, from_node)
		}
	}
}

func (n *terminalN) SendPacket(destinationMac string, destinationIp string, data string, headerSize int) {
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

		packet := packet.NewFragment(n.macAddress, destinationMac, n.ipAddress, destinationIp, 64, headerSize, payloadSize, n.nes.CurrentTime, originalDataId, moreFragment, fragmentOffset, fragmentData)

		n.sendPacket(packet) // 細かくfragmentにしたpacketを送信する
		offset += payloadSize
	}
}

func (n *terminalN) createPacket(destinationMac string, destinationIp string, headerSize int, payloadSize int, payload string) {
	p := packet.NewPacket(n.macAddress, destinationMac, n.ipAddress, destinationIp, 64, headerSize, payloadSize, n.nes.CurrentTime, payload)
	n.nes.LogPacketInfo(p, "created", n.nodeId)
	n.SendPacket(destinationMac, destinationIp, payload, headerSize)
}

func (n *terminalN) SetTraffic(destinationMac string, destinationIp string, bitrate float64, startTime float64, duration float64, headerSize int, payloadSize int, burstiness float64) {
	endTime := startTime + duration
	packetSize := headerSize + payloadSize
	// burstinessはよくわからん
	// このintervalで送れば，理論上指定したbitrateになる．
	interval := float64(packetSize*8) / bitrate * burstiness

	// 全部のcreatePacketのスケジュールを最初にしておく
	for t := startTime; t < endTime; t += interval {
		n.nes.ScheduleEvent(t, func(args ...any) {
			n.createPacket(destinationMac, destinationIp, headerSize, payloadSize, strings.Repeat("X", payloadSize))
		})
	}
}
