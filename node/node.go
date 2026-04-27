package node

import (
	"container/heap"
	"fmt"
	"math/rand"
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

// Linkの構造体

// linkのqueueに突っ込むパケットとか
type PacketWithQueueTime struct {
	dequeTime float64
	packet    *packet.Packet
	fromNode  *N
}

type LinkQueue []*PacketWithQueueTime

func (lq LinkQueue) Len() int { return len(lq) }

func (lq LinkQueue) Less(i, j int) bool {
	return lq[i].dequeTime < lq[j].dequeTime
}

func (lq LinkQueue) Swap(i, j int) { lq[i], lq[j] = lq[j], lq[i] }

func (lq *LinkQueue) Push(x any) {
	*lq = append(*lq, x.(*PacketWithQueueTime))
}

func (lq *LinkQueue) Pop() any {
	old := *lq
	n := len(old)
	item := old[n-1]
	*lq = old[:n-1]
	return item
}

type Link struct {
	node_x             *N
	node_y             *N
	bandwidth          float64
	delay              float64
	packet_loss        float64
	nes                *nteventsched.NtEventSched
	packetQueueXY      LinkQueue
	packetQueueYX      LinkQueue
	currentQueueTimeXY float64
	currentQueueTimeYX float64
}

func NewLink(node_x *N, node_y *N, bandwidth float64, delay float64, packet_loss float64, nes *nteventsched.NtEventSched) *Link {
	nes.AddEdge(node_x.nodeId, node_y.nodeId, fmt.Sprintf("%v Mbps %v s\n", bandwidth/1000000, delay), bandwidth, delay)
	l := Link{node_x: node_x, node_y: node_y, bandwidth: bandwidth, delay: delay, packet_loss: packet_loss, nes: nes}
	node_x.addLink(&l)
	node_y.addLink(&l)
	return &l
}

func (l *Link) PrintLink() {
	fmt.Printf("%v <-> %v 帯域幅: %v 遅延: %v パケットロス率：%v\n", l.node_x.nodeId, l.node_y.nodeId, l.bandwidth, l.delay, l.packet_loss)
}

// リンクを通してfrom_nodeからto_nodeにパケットを転送する関数
func (l *Link) transferPacket(from_node *N) {
	var queue *LinkQueue
	var nextNode *N
	if l.node_x.nodeId != from_node.nodeId {
		queue = &l.packetQueueYX
		nextNode = l.node_x
	} else {
		queue = &l.packetQueueXY
		nextNode = l.node_y
	}

	if queue.Len() != 0 {
		item := heap.Pop(queue).(*PacketWithQueueTime)
		dequeTime := item.dequeTime
		p := item.packet
		packetTransferTime := (p.Size * 8) / l.bandwidth

		// パケットロス
		if rand.Intn(100) < int(l.packet_loss*100) {
			p.SetArrived(-1)
		}

		// currentTime + delayの時間から，nextNodeがpacketを受け取り始める
		l.nes.ScheduleEvent(l.nes.CurrentTime+l.delay, func(args ...any) {
			nextNode.receivePacket(args[0].(*packet.Packet))
		}, p)

		// dequeTime(currentTime) + packetTransferTimeで，完全にpacketをlinkに流し終えるので，queueの待ち時間から引ける．
		l.nes.ScheduleEvent(dequeTime+packetTransferTime, func(args ...any) {
			l.subtractFromQueueTime(args[0].(*N), args[1].(float64))
		}, from_node, packetTransferTime)

		if queue.Len() != 0 { // 次のパケットがある場合
			nextPacket := (*queue)[0]
			l.nes.ScheduleEvent(nextPacket.dequeTime, func(args ...any) {
				l.transferPacket(args[0].(*N))
			}, from_node)
		}
	}
}

func (l *Link) enqueuePacket(pkt *packet.Packet, from_node *N) {
	var currentQueueTime float64
	var queue *LinkQueue
	if l.node_x.nodeId != from_node.nodeId {
		currentQueueTime = l.currentQueueTimeYX
		queue = &l.packetQueueYX
	} else {
		currentQueueTime = l.currentQueueTimeXY
		queue = &l.packetQueueXY
	}

	packetTransferTime := pkt.Size * 8 / l.bandwidth
	dequeTime := l.nes.CurrentTime + currentQueueTime
	heap.Push(queue, &PacketWithQueueTime{dequeTime: dequeTime, packet: pkt, fromNode: from_node})
	l.addToQueueTime(from_node, packetTransferTime)

	if queue.Len() == 1 {
		l.nes.ScheduleEvent(dequeTime, func(args ...any) {
			l.transferPacket(args[0].(*N))
		}, from_node)
	}
}

func (l *Link) addToQueueTime(from_node *N, packetTransferTime float64) {
	if l.node_x.nodeId != from_node.nodeId {
		l.currentQueueTimeYX += packetTransferTime
	} else {
		l.currentQueueTimeXY += packetTransferTime
	}
}

func (l *Link) subtractFromQueueTime(from_node *N, packetTransferTime float64) {
	if l.node_x.nodeId != from_node.nodeId {
		l.currentQueueTimeYX -= packetTransferTime
	} else {
		l.currentQueueTimeXY -= packetTransferTime
	}
}
