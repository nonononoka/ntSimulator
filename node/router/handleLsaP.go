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

func (r *router) getLinkStateInfos() map[string]packet.LinkStateInfo {
	linkStateInfos := make(map[string]packet.LinkStateInfo)
	for l, ipAddress := range r.interfaces {
		linkStateInfos[l.GetId()] = packet.LinkStateInfo{IpAddress: ipAddress.String(), Cost: l.GetLinkCost()}
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
	linkStateInfos := make(map[string]packet.LinkStateInfo)
	for link, ipAddress := range r.interfaces {
		linkStateInfos[link.GetId()] = packet.LinkStateInfo{IpAddress: ipAddress.String(), Cost: link.GetLinkCost()}
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
		for linkKey, info := range entry.linkStateInfos {
			fmt.Printf("  - Link [%s]: IP Address = %s, Cost = %.2f\n",
				linkKey, info.IpAddress, info.Cost)
		}
	}

	fmt.Println("=======================================")
}

func (r *router) calculateShortestPaths(startRouterId int) {

}
