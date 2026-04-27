package network

import (
	"fmt"
	"nt-simulator/nteventsched"
	"nt-simulator/packet"
)

// Nodeの構造体
type N struct {
	nodeId  int
	address string
	links   []*Link
	nes     *nteventsched.NtEventSched
}

func (n *N) Address() string {
	return n.address
}

func (n *N) PrintNode() {
	connected_nodes := make([]int, 0, 10)
	for _, v := range n.links {
		if v.node_x.nodeId != n.nodeId {
			connected_nodes = append(connected_nodes, v.node_x.nodeId)
		}
		if v.node_y.nodeId != n.nodeId {
			connected_nodes = append(connected_nodes, v.node_y.nodeId)
		}
	}
	fmt.Printf("ノード(ID: %v, アドレス: %s), 接続ノード: %v\n", n.nodeId, n.address, connected_nodes)
}

func NewNode(node_id int, address string, nes *nteventsched.NtEventSched) *N {
	nes.AddNode(node_id)
	return &N{nodeId: node_id, address: address, nes: nes}
}

func (n *N) addLink(link *Link) {
	for _, l := range n.links {
		if l == link {
			return
		}
	}
	n.links = append(n.links, link)
}

func (n *N) SendPacket(p *packet.Packet) {
	n.nes.LogPacketInfo(p, "sent", n.nodeId)
	if p.Header.Destination == n.address {
		n.receivePacket(p)
	} else {
		for _, l := range n.links {
			var from_node *N = n
			l.enqueuePacket(p, from_node)
			break
		}
	}
}

func (n *N) receivePacket(p *packet.Packet) {
	if p.ArrivalTime() == -1 {
		n.nes.LogPacketInfo(p, "lost", n.nodeId)
		return
	}
	if p.Header.Destination == n.address {
		n.nes.LogPacketInfo(p, "arrived", n.nodeId)
		p.SetArrived(n.nes.CurrentTime)
	} else {
		n.nes.LogPacketInfo(p, "received", n.nodeId)
		// 宛先が自分自身にない場合
	}
}

func (n *N) createPacket(destination string, headerSize float64, payloadSize float64) {
	p := packet.NewPacket(n.address, destination, headerSize, payloadSize, n.nes.CurrentTime)
	n.nes.LogPacketInfo(p, "created", n.nodeId)
	n.SendPacket(p)
}

func (n *N) SetTraffic(destination string, bitrate float64, startTime float64, duration float64, headerSize float64, payloadSize float64, burstiness float64) {
	endTime := startTime + duration
	packetSize := headerSize + payloadSize
	// burstinessはよくわからん
	// このintervalで送れば，理論上指定したbitrateになる．
	interval := (packetSize * 8) / bitrate * burstiness

	// 全部のcreatePacketのスケジュールを最初にしておく
	for t := startTime; t < endTime; t += interval {
		n.nes.ScheduleEvent(t, func(args ...any) {
			n.createPacket(destination, headerSize, payloadSize)
		})
	}
}
