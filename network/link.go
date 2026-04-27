package network

import (
	"container/heap"
	"fmt"
	"math/rand"
	"nt-simulator/nteventsched"
	"nt-simulator/packet"
)

// linkのqueueに突っ込むパケットとか
type PacketWithQueueTime struct {
	dequeTime float64
	packet    *packet.Packet
	fromNode  node
}

type LinkQueue []*PacketWithQueueTime

type Link struct {
	node_x             node
	node_y             node
	bandwidth          float64
	delay              float64
	packet_loss        float64
	nes                *nteventsched.NtEventSched
	packetQueueXY      LinkQueue
	packetQueueYX      LinkQueue
	currentQueueTimeXY float64
	currentQueueTimeYX float64
}

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

func NewLink(node_x node, node_y node, bandwidth float64, delay float64, packet_loss float64, nes *nteventsched.NtEventSched) *Link {
	nes.AddEdge(node_x.NodeId(), node_y.NodeId(), fmt.Sprintf("%v Mbps %v s\n", bandwidth/1000000, delay), bandwidth, delay)
	l := Link{node_x: node_x, node_y: node_y, bandwidth: bandwidth, delay: delay, packet_loss: packet_loss, nes: nes}
	node_x.AddLink(&l)
	node_y.AddLink(&l)
	return &l
}

func (l *Link) PrintLink() {
	fmt.Printf("%v <-> %v 帯域幅: %v 遅延: %v パケットロス率：%v\n", l.node_x.NodeId(), l.node_y.NodeId(), l.bandwidth, l.delay, l.packet_loss)
}

// リンクを通してfrom_nodeからto_nodeにパケットを転送する関数
func (l *Link) transferPacket(from_node node) {
	var queue *LinkQueue
	var nextNode node
	if l.node_x.NodeId() != from_node.NodeId() {
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
			l.subtractFromQueueTime(args[0].(node), args[1].(float64))
		}, from_node, packetTransferTime)

		if queue.Len() != 0 { // 次のパケットがある場合
			nextPacket := (*queue)[0]
			l.nes.ScheduleEvent(nextPacket.dequeTime, func(args ...any) {
				l.transferPacket(args[0].(node))
			}, from_node)
		}
	}
}

func (l *Link) enqueuePacket(pkt *packet.Packet, from_node node) {
	var currentQueueTime float64
	var queue *LinkQueue
	if l.node_x.NodeId() != from_node.NodeId() {
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
			l.transferPacket(args[0].(node))
		}, from_node)
	}
}

func (l *Link) addToQueueTime(from_node node, packetTransferTime float64) {
	if l.node_x.NodeId() != from_node.NodeId() {
		l.currentQueueTimeYX += packetTransferTime
	} else {
		l.currentQueueTimeXY += packetTransferTime
	}
}

func (l *Link) subtractFromQueueTime(from_node node, packetTransferTime float64) {
	if l.node_x.NodeId() != from_node.NodeId() {
		l.currentQueueTimeYX -= packetTransferTime
	} else {
		l.currentQueueTimeXY -= packetTransferTime
	}
}
