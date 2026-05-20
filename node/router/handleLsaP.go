package router

import (
	"fmt"
	"math/rand/v2"
	"nt-simulator/address"
	"nt-simulator/link"
	"nt-simulator/packet"
)

func (r *router) scheduleLsaPacket() {
	randomDelay := 0.3 + rand.Float64()*0.2
	r.GetNES().ScheduleEvent(r.GetNES().CurrentTime+randomDelay, func(args ...any) { r.sendLsaPacket() })
}

func (r *router) sendLsaPacket() {
	seqNumber := r.incrementLsaSequence()
	linkStateInfos := r.getLinkStateInfos()

	for link, ipAddress := range r.interfaces {
		lsaP := packet.NewLsaP(
			address.NewMacAddress("00:00:00:00:00:00"), ipAddress, r.GetNES().CurrentTime, r.NodeId(), seqNumber, linkStateInfos)
		link.EnqueuePacket(lsaP, r)
	}

	r.GetNES().ScheduleEvent(r.GetNES().CurrentTime+r.lsaInterval, func(args ...any) {
		r.sendLsaPacket()
	})
}

func (r *router) getLinkStateInfos() []packet.LinkStateInfo {
	linkStateInfos := make([]packet.LinkStateInfo, 0, len(r.interfaces))
	for l, ipAddress := range r.interfaces {
		linkStateInfos = append(linkStateInfos, packet.LinkStateInfo{
			NodeXId:   l.NodeX().NodeId(),
			NodeYId:   l.NodeY().NodeId(),
			IpAddress: ipAddress.String(),
			Cost:      l.GetLinkCost(),
		})
	}
	return linkStateInfos
}

func (r *router) incrementLsaSequence() int {
	r.lsaSequenceNumber += 1
	return r.lsaSequenceNumber
}

func (r *router) floodLsa(p *packet.LsaP) {
	// 受信したLSAパケットを他のルーターにフラッドする
	lp, err := p.ParsePayload()
	if err != nil {
		fmt.Printf("Lsa parse error: %v\n", err)
		return
	}
	routerId := lp.RouterId

	for link := range r.interfaces {
		if link.NodeX().NodeId() != routerId && link.NodeY().NodeId() != routerId {
			link.EnqueuePacket(p, r)
		}
	}
}

func (r *router) receiveLsaPacket(p *packet.LsaP, receivedLink *link.Link) {
	lp, err := p.ParsePayload()
	if err != nil {
		fmt.Printf("Lsa parse error: %v\n", err)
		return
	}

	if !r.isTopologyInitialized {
		r.isTopologyInitialized = true
		r.initializeTopologyDatabase()
	}

	routerId := lp.RouterId
	lsaInfo := lp.LinkStateInfos
	seqNumber := lp.SequenceNumber
	now := r.GetNES().CurrentTime

	var currentLsaInfo topologyEntry
	if _, ok := r.topologyDatabase[routerId]; !ok {
		currentLsaInfo = topologyEntry{}
	} else {
		currentLsaInfo = r.topologyDatabase[routerId]
	}

	// 受信したLSAが新しい情報を持っている場合、トポロジーデータベースを更新
	if seqNumber > currentLsaInfo.sequenceNumber {
		r.topologyDatabase[routerId] = topologyEntry{
			sequenceNumber: seqNumber,
			linkStateInfos: lsaInfo,
		}
		if r.GetNES().Verbose {
			r.printTopologyDatabase()
		}
		r.floodLsa(p)
	} else {
		fmt.Printf("%v 古いLSAを受信しました %v\n", now, r.NodeId())
	}
}

func (r *router) initializeTopologyDatabase() {
	linkStateInfos := make([]packet.LinkStateInfo, 0, len(r.interfaces))
	for l, ipAddress := range r.interfaces {
		linkStateInfos = append(linkStateInfos, packet.LinkStateInfo{
			NodeXId:   l.NodeX().NodeId(),
			NodeYId:   l.NodeY().NodeId(),
			IpAddress: ipAddress.String(),
			Cost:      l.GetLinkCost(),
		})
	}

	r.topologyDatabase[r.NodeId()] = topologyEntry{sequenceNumber: 0, linkStateInfos: linkStateInfos}
}

func (r *router) printTopologyDatabase() {
	fmt.Printf("========== TOPOLOGY DATABASE ==========（ルーター:%v）\n", r.NodeId())
	for routerID, entry := range r.topologyDatabase {
		fmt.Printf("Router ID: %d (Seq: %d)\n", routerID, entry.sequenceNumber)

		if len(entry.linkStateInfos) == 0 {
			fmt.Println("  [No Link State Information]")
			continue
		}

		// 各リンクの情報をループして出力
		for _, info := range entry.linkStateInfos {
			fmt.Printf("  - Link [%v <-> %v]: IP Address = %s, Cost = %.2f\n",
				info.NodeXId, info.NodeYId, info.IpAddress, info.Cost)
		}
	}

	fmt.Println("=======================================")
}

type gNode struct {
	cost   int
	nodeId int
}

type heapq []*gNode

func (lq heapq) Len() int { return len(lq) }

func (lq heapq) Less(i, j int) bool {
	return lq[i].cost < lq[j].cost
}

func (lq heapq) Swap(i, j int) { lq[i], lq[j] = lq[j], lq[i] }

func (lq *heapq) Push(x any) {
	*lq = append(*lq, x.(*gNode))
}

func (lq *heapq) Pop() *gNode {
	old := *lq
	n := len(old)
	item := old[n-1]
	*lq = old[:n-1]
	return item
}

// func (r *router) calculateShortestPaths(startRouterId int) {
// 	shortestPaths = make(map[int]float64)
// 	previousNodes := make(map[int]int) // 各ルーターの前に辿るルーターを記録
// 	for r := range r.topologyDatabase {
// 		shortestPaths[r] = math.Inf(1)
// 	}
// 	shortestPaths[startRouterId] = 0
// 	queue := &heapq{&gNode{cost: 0, nodeId: startRouterId}}

// 	for queue.Len() != 0 {
// 		currentNode := queue.Pop()

// 		for link, linkInfo := range r.topologyDatabase[currentNode.nodeId].linkStateInfos {
// 			// このlinkからいけるrouterを
// 		}
// 	}
// }
