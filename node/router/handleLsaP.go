package router

import (
	"fmt"
	"math"
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
			address.ZeroMacAddress, ipAddress, r.GetNES().CurrentTime, r.NodeId(), seqNumber, linkStateInfos)
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
		if link.NodeX().NodeId() != routerId && link.NodeY().NodeId() != routerId { // 隣人の情報じゃなかったらflood
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
		// ルーティングテーブルの再計算
		r.updateRoutingTableWithDijkstra()

		// LSAを近接ルータに再送信
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
	// fmt.Printf("========== TOPOLOGY DATABASE ==========（ルーター:%v）\n", r.NodeId())
	// for routerID, entry := range r.topologyDatabase {
	// 	fmt.Printf("Router ID: %d (Seq: %d)\n", routerID, entry.sequenceNumber)

	// 	if len(entry.linkStateInfos) == 0 {
	// 		fmt.Println("  [No Link State Information]")
	// 		continue
	// 	}

	// 	// 各リンクの情報をループして出力
	// 	for _, info := range entry.linkStateInfos {
	// 		fmt.Printf("  - Link [%v <-> %v]: IP Address = %s, Cost = %.2f\n",
	// 			info.NodeXId, info.NodeYId, info.IpAddress, info.Cost)
	// 	}
	// }

	// fmt.Println("=======================================")
}

type gNode struct {
	cost   float64
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

func (r *router) calculateShortestPaths(startRouterId int) (map[int]float64, map[int]int) {
	shortestPaths := make(map[int]float64) // 各ルータごとのstartRouterIdからのcostを記録
	previousNodes := make(map[int]int)     // 各ルーターの前に辿るルーターを記録
	for r := range r.topologyDatabase {
		shortestPaths[r] = math.Inf(1)
	}
	shortestPaths[startRouterId] = 0
	queue := &heapq{&gNode{cost: 0, nodeId: startRouterId}}

	for queue.Len() != 0 {
		currentNode := queue.Pop()

		for _, linkStateInfo := range r.topologyDatabase[currentNode.nodeId].linkStateInfos {
			// このlinkで直接繋がっているrouterたちのshortestPathを更新して、更新されたらqueueに突っ込む
			neighborId := getNeighorRouterId(currentNode.nodeId, linkStateInfo.NodeXId, linkStateInfo.NodeYId)
			if neighborId == -1 {
				panic("このlinkはcurrentNodeと繋がっていません")
			}
			if _, ok := r.topologyDatabase[neighborId]; ok {
				newCost := currentNode.cost + linkStateInfo.Cost
				if newCost < shortestPaths[neighborId] {
					shortestPaths[neighborId] = newCost
					previousNodes[neighborId] = currentNode.nodeId
					queue.Push(&gNode{cost: newCost, nodeId: neighborId})
				}
			}
		}
	}

	return shortestPaths, previousNodes
}

func (r *router) updateRoutingTableWithDijkstra() {

	shortestPaths, previousNodes := r.calculateShortestPaths(r.NodeId())
	tmpRoutingTable := make(map[address.IpAddress]*routingTableEntry)

	// 各routerに行くまでに最初にどこに行ったらいいかをupdateする
	for destination := range shortestPaths {
		if destination != r.NodeId() {
			if math.IsInf(shortestPaths[destination], 1) {
				continue // LSA未到達で到達不可能なノードはスキップ
			}
			// このrouterからdestinationに行くにはまずnextHopRouterIdに行く必要がある
			nextHopRouterId := findInitialHop(destination, previousNodes, r.NodeId())
			linkToNextHop := r.getLinkToNeighbor(nextHopRouterId)

			for _, linkStateInfo := range r.topologyDatabase[destination].linkStateInfos {
				destinationIPAddress := address.NewIPAddress(linkStateInfo.IpAddress) // destinationのrouterが持っているipaddress
				// rとdestinationが同じnetworkだったら、hopは不要になるので-1にする
				nextHop := nextHopRouterId
				for _, ip := range r.interfaces {
					if ip.IsSameNetwork(destinationIPAddress) {
						nextHop = -1
					}
				}
				tmpRoutingTable[*destinationIPAddress.ConvertToNetworkCIDR()] = &routingTableEntry{nexthop: nextHop, link: linkToNextHop}
			}
		}
	}

	// 自分のinterfaceに接続されているネットワークへのルートを追加
	for l, ipAddress := range r.interfaces {
		if _, ok := tmpRoutingTable[*ipAddress.ConvertToNetworkCIDR()]; !ok {
			tmpRoutingTable[*ipAddress.ConvertToNetworkCIDR()] = &routingTableEntry{nexthop: -1, link: l}
		}
	}

	clear(r.routingTable)
	for ip, entry := range tmpRoutingTable {
		r.routingTable[&ip] = *entry
	}

	PrintRoutingTable(r.NodeId(), r.routingTable)
}

func PrintRoutingTable(nodeId int, table map[*address.IpAddress]routingTableEntry) {
	fmt.Printf("--- ROUTING TABLE LOG START --- %v\n", nodeId)

	if len(table) == 0 {
		fmt.Println("  (Table is empty)")
	}

	for ip, entry := range table {
		ipStr := "nil"
		if ip != nil {
			ipStr = ip.String()
		}

		// 3. ログに1行で出力
		fmt.Printf("DestIP: %-15s -> NextHop(RouterID): %-3d | Via Link: %v <-> %v\n",
			ipStr,
			entry.nexthop,
			entry.link.NodeX().NodeId(),
			entry.link.NodeY().NodeId(),
		)
	}

	fmt.Println("--- ROUTING TABLE LOG END ---")
}

func (r *router) getLinkToNeighbor(neighborRouterId int) *link.Link {
	return r.neighbors[neighborRouterId].link
}

func getNeighorRouterId(currentRouterId int, nodeXId int, nodeYId int) int {
	switch currentRouterId {
	case nodeXId:
		return nodeYId
	case nodeYId:
		return nodeXId
	default:
		return -1
	}
}

// startRouterIdからdestinationに行くまでの、startRouterIdの次のhop
func findInitialHop(desitination int, previousNodes map[int]int, startRouterId int) int {
	currentRouterId := desitination
	for {
		p, ok := previousNodes[currentRouterId]
		if !ok {
			break // 親ノードがいなくなったら（行き止まり）ループを抜ける
		}

		if p == startRouterId {
			return currentRouterId // スタートの直前まで戻ってきたので、その時のノードを返す
		}
		currentRouterId = p
	}

	panic(fmt.Sprintf("no valid path from start %v to destination %v", startRouterId, desitination))
}
