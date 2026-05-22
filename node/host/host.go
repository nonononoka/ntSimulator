package host

import (
	"fmt"
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

// Nodeの構造体
type host struct {
	*basenode.BaseNode
	mtu               int
	fragmentedPackets map[string]map[int]packetI.PacketI
	*address.MacAddress
	*address.IpAddress
	arrivedCount  int
	receivedBytes int
}

func (n *host) ArrivedCount() int  { return n.arrivedCount }
func (n *host) ReceivedBytes() int { return n.receivedBytes }

func NewHost(nodeId int, ipAddress string, mtu int, nes *nteventsched.NtEventSched) *host {
	n := &host{BaseNode: basenode.NewBaseNode(nodeId, nes), fragmentedPackets: make(map[string]map[int]packetI.PacketI), mtu: mtu, MacAddress: address.NewMacAddress(address.GenerateRandomMAC()),
		IpAddress: address.NewIPAddress(ipAddress)}
	nes.AddNode(n)
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
	if p.GetHeader().DestinationMac.String() == n.MacAddress.String() && p.GetHeader().DestIp.String() == n.IpAddress.String() {
		n.GetNES().LogPacketInfo(p, "arrived", n.NodeId())
		p.SetArrived(n.GetNES().CurrentTime)
		n.arrivedCount++
		n.receivedBytes += len(p.GetPayload())

		if p.GetHeader().FragmentFlags.MoreFragment {
			n.storeFragment(p)
		} else if p.GetHeader().FragmentOffset > 0 {
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

func (n *host) SetTraffic(destinationMac string, destinationIp string, bitrate float64, startTime float64, duration float64, headerSize int, payloadSize int, burstiness float64) {
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
	n.GetNES().ScheduleEvent(startTime, func(args ...any) {
		n.createPacket(address.NewMacAddress(destinationMac), address.NewIPAddress(destinationIp), headerSize, payloadSize, strings.Repeat("X", payloadSize))
	})
}

func (n *host) GetIPAddresses() []*address.IpAddress {
	return []*address.IpAddress{n.IpAddress}
}

// fragmentedPacketsにoriginalDataIdのところにoffset付きで保管する
func (n *host) storeFragment(fragment packetI.PacketI) {
	originalDataId := fragment.GetHeader().FragmentFlags.OriginalDataId
	offset := fragment.GetHeader().FragmentOffset

	if _, ok := n.fragmentedPackets[originalDataId]; !ok {
		n.fragmentedPackets[originalDataId] = make(map[int]packetI.PacketI)
	}

	n.fragmentedPackets[originalDataId][offset] = fragment
	n.GetNES().LogPacketInfo(fragment, fmt.Sprintf("fragment_stored offset:%v originalDataId:%s moreflagment:%v", fragment.GetHeader().FragmentOffset, fragment.GetHeader().FragmentFlags.OriginalDataId, fragment.GetHeader().FragmentFlags.MoreFragment), n.NodeId())
}

func (n *host) reassembleAndProcessPacket(lastFragment packetI.PacketI) {
	originalDataId := lastFragment.GetHeader().FragmentFlags.OriginalDataId
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

	expectedLength := lastFragment.GetHeader().FragmentOffset + len(lastFragment.GetPayload())
	if len(assembledPayload) != expectedLength {
		n.GetNES().LogPacketInfo(lastFragment, fmt.Sprintf("reassemble failed: missing fragments (expected %d bytes, got %d bytes)", expectedLength, len(assembledPayload)), n.NodeId())
		return
	}
	n.GetNES().LogPacketInfo(lastFragment, "reassembled", n.NodeId())
}

func (n *host) internalSendPacket(p *packet.Packet) {
	n.GetNES().LogPacketInfo(p, "sent", n.NodeId())
	if p.Header.DestinationMac == n.MacAddress {
		n.ReceivePacket(p, nil)
	} else {
		for _, l := range n.GetLinks() {
			var from_node *host = n
			l.EnqueuePacket(p, from_node)
		}
	}
}

func (n *host) sendPacket(destinationMac *address.MacAddress, destinationIp *address.IpAddress, data string, headerSize int) {
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

		packet := packet.NewFragment(n.MacAddress, destinationMac, n.IpAddress, destinationIp, 64, headerSize, n.GetNES().CurrentTime, originalDataId, moreFragment, fragmentOffset, fragmentData)

		n.internalSendPacket(packet) // 細かくfragmentにしたpacketを送信する
		offset += payloadSize
	}
}

func (n *host) createPacket(destinationMac *address.MacAddress, destinationIp *address.IpAddress, headerSize int, payloadSize int, payload string) {
	p := packet.NewPacket(n.MacAddress, destinationMac, n.IpAddress, destinationIp, 64, headerSize, payloadSize, n.GetNES().CurrentTime, payload)
	n.GetNES().LogPacketInfo(p, "created", n.NodeId())
	n.sendPacket(destinationMac, destinationIp, payload, headerSize)
}
