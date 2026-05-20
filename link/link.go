package link

import (
	"container/heap"
	"fmt"
	"math"
	"math/rand"
	"nt-simulator/address"
	"nt-simulator/nteventsched"
	"nt-simulator/packet/packetI"

	"github.com/google/uuid"
)

type Link struct {
	id                 string
	nodeX              node
	nodeY              node
	bandwidth          float64
	delay              float64
	packetLoss         float64
	nes                *nteventsched.NtEventSched
	packetQueueXY      linkq
	packetQueueYX      linkq
	currentQueueTimeXY float64
	currentQueueTimeYX float64
}

// networkに含まれるnode（hostとかswitchとか含めて）のinterface
type node interface {
	NodeId() int
	AddLink(link *Link, ip *address.IpAddress)
	ReceivePacket(p packetI.PacketI, l *Link)
	GetIPAddresses() []*address.IpAddress
}

// linkのqueueに突っ込むパケットとか
type packetWithQueueTime struct {
	dequeTime float64
	packet    packetI.PacketI
	fromNode  node
}

type linkq []*packetWithQueueTime

func NewLink(nodeX node, nodeY node, bandwidth float64, delay float64, packetLoss float64, nes *nteventsched.NtEventSched) *Link {
	ipX, ipY := setupLinkIps(nodeX, nodeY)
	fmt.Printf("%v %v\n", nodeX.NodeId(), nodeY.NodeId())
	nes.AddEdge(nodeX.NodeId(), nodeY.NodeId(), fmt.Sprintf("%v Mbps %v s\n", bandwidth/1000000, delay), bandwidth, delay)
	l := Link{id: uuid.New().String(), nodeX: nodeX, nodeY: nodeY, bandwidth: bandwidth, delay: delay, packetLoss: packetLoss, nes: nes}
	nodeX.AddLink(&l, ipX)
	nodeY.AddLink(&l, ipY)
	return &l
}

func (lq linkq) Len() int { return len(lq) }

func (lq linkq) Less(i, j int) bool {
	return lq[i].dequeTime < lq[j].dequeTime
}

func (lq linkq) Swap(i, j int) { lq[i], lq[j] = lq[j], lq[i] }

func (lq *linkq) Push(x any) {
	*lq = append(*lq, x.(*packetWithQueueTime))
}

func (lq *linkq) Pop() any {
	old := *lq
	n := len(old)
	item := old[n-1]
	*lq = old[:n-1]
	return item
}

func (l *Link) GetId() string {
	return l.id
}

func (l *Link) PrintLink() {
	fmt.Printf("%v <-> %v 帯域幅: %v 遅延: %v パケットロス率：%v\n", l.nodeX.NodeId(), l.nodeY.NodeId(), l.bandwidth, l.delay, l.packetLoss)
}

func (l *Link) NodeX() node {
	return l.nodeX
}

func (l *Link) NodeY() node {
	return l.nodeY
}

func (l *Link) EnqueuePacket(pkt packetI.PacketI, from_node node) {
	var currentQueueTime float64
	var queue *linkq
	if l.nodeX.NodeId() != from_node.NodeId() {
		currentQueueTime = l.currentQueueTimeYX
		queue = &l.packetQueueYX
	} else {
		currentQueueTime = l.currentQueueTimeXY
		queue = &l.packetQueueXY
	}

	packetTransferTime := float64(pkt.GetSize()*8) / l.bandwidth
	dequeTime := l.nes.CurrentTime + currentQueueTime
	heap.Push(queue, &packetWithQueueTime{dequeTime: dequeTime, packet: pkt, fromNode: from_node})
	l.addToQueueTime(from_node, packetTransferTime)

	if queue.Len() == 1 {
		l.nes.ScheduleEvent(dequeTime, func(args ...any) {
			l.transferPacket(args[0].(node))
		}, from_node)
	}
}

func (l *Link) GetLinkCost() float64 {
	minCost := 0.000000001
	return math.Max(minCost, 1.0/l.bandwidth)
}

// リンクを通してfrom_nodeからto_nodeにパケットを転送する関数
func (l *Link) transferPacket(from_node node) {
	var queue *linkq
	var nextNode node
	if l.nodeX.NodeId() != from_node.NodeId() {
		queue = &l.packetQueueYX
		nextNode = l.nodeX
	} else {
		queue = &l.packetQueueXY
		nextNode = l.nodeY
	}

	if queue.Len() != 0 {
		item := heap.Pop(queue).(*packetWithQueueTime)
		dequeTime := item.dequeTime
		p := item.packet
		packetTransferTime := float64(p.GetSize()*8) / l.bandwidth

		// パケットロス
		if rand.Intn(100) < int(l.packetLoss*100) {
			p.SetArrived(-1)
		}

		// currentTime + delayの時間から，nextNodeがpacketを受け取り始める
		l.nes.ScheduleEvent(l.nes.CurrentTime+l.delay, func(args ...any) {
			nextNode.ReceivePacket(args[0].(packetI.PacketI), l)
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

func (l *Link) addToQueueTime(from_node node, packetTransferTime float64) {
	if l.nodeX.NodeId() != from_node.NodeId() {
		l.currentQueueTimeYX += packetTransferTime
	} else {
		l.currentQueueTimeXY += packetTransferTime
	}
}

func (l *Link) subtractFromQueueTime(from_node node, packetTransferTime float64) {
	if l.nodeX.NodeId() != from_node.NodeId() {
		l.currentQueueTimeYX -= packetTransferTime
	} else {
		l.currentQueueTimeXY -= packetTransferTime
	}
}

func setupLinkIps(nodeX node, nodeY node) (*address.IpAddress, *address.IpAddress) {
	// ノードから利用可能なIPアドレスリストを取得
	ipListX := getAvailableIPList(nodeX)
	ipListY := getAvailableIPList(nodeY)

	// 互換性のあるIPアドレスを選択
	selectedIPX, selectedIPY := selectCompatibleIp(ipListX, ipListY)

	// 使用済みIPアドレスにフラグを設定
	return selectedIPX, selectedIPY
}

func getAvailableIPList(node node) []*address.IpAddress {
	return node.GetIPAddresses()
}

func selectCompatibleIp(ipListX []*address.IpAddress, ipListY []*address.IpAddress) (*address.IpAddress, *address.IpAddress) {
	for _, ipCIDRX := range ipListX {
		for _, ipCIDRY := range ipListY {
			if ipCIDRX.IsSameNetwork(ipCIDRY) {
				return ipCIDRX, ipCIDRY
			}
		}
	}
	panic("互換性のあるipアドレスのペアが見つかりませんでした")
}
