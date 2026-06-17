package router

import (
	"fmt"
	"math/rand/v2"
	"nt-simulator/address"
	"nt-simulator/link"
	"nt-simulator/packet"
)

// helloパケット関連

func (r *router) scheduleHelloPacket() {
	randomDelay := rand.Float64() * 0.1
	r.GetNES().ScheduleEvent(r.GetNES().CurrentTime+randomDelay, func(args ...any) { r.sendHelloPacket() })
}

func (r *router) sendHelloPacket() {
	var neighbors []int
	for neighborRouterId := range r.neighbors {
		neighbors = append(neighbors, neighborRouterId)
	}

	for l, interfaceCIDR := range r.interfaces {
		helloP := packet.NewHelloP(address.ZeroMacAddress, interfaceCIDR, r.GetNES().CurrentTime, r.NodeId(), r.helloInterval, neighbors)
		l.EnqueuePacket(helloP, r)
	}
	r.GetNES().ScheduleEvent(r.GetNES().CurrentTime+r.helloInterval, func(args ...any) {
		r.sendHelloPacket()
	})
}

func (r *router) receiveHelloPacket(p *packet.HelloP, receivedLink *link.Link) {
	hp, err := p.ParsePayload()
	if err != nil {
		fmt.Printf("Hello parse error: %v\n", err)
		return
	}
	routerId := hp.RouterId
	newNeighbor := false
	now := r.GetNES().CurrentTime

	if _, ok := r.neighbors[routerId]; !ok {
		// 未知のルーターなので、新しい隣接情報を更新
		newNeighbor = true
		r.neighbors[routerId] = &neighborInfo{
			lastHelloTime: now,
			link:          receivedLink,
			info:          hp,
		}
	} else {
		// 既知のルーターの隣接情報を更新
		lastHelloTime := r.neighbors[routerId].lastHelloTime
		if now > lastHelloTime {
			r.neighbors[routerId].lastHelloTime = now
		}
		if receivedLink != r.neighbors[routerId].link {
			newNeighbor = true
			r.neighbors[routerId].link = receivedLink
		}
		if !hp.Equals(r.neighbors[routerId].info) {
			newNeighbor = true
			r.neighbors[routerId].info = hp
		}
	}
	if r.GetNES().Verbose {

		if newNeighbor {
			r.printNeighborInfo()
		} else {
			fmt.Printf("%v Helloパケットを受信しましたが、隣接ルーターの情報は更新されていません。ルーターID: %v \n", r.GetNES().CurrentTime, r.NodeId())
		}
	}
}

func (r *router) printNeighborInfo() {
	for routerId, neighborInfo := range r.neighbors {
		fmt.Printf("ルーターID: %v \n", routerId)
		fmt.Printf("最後のhello受信時刻: %v\n", neighborInfo.lastHelloTime)
		fmt.Printf("隣接ルーターへのリンク：リンク %v <-> %v\n", neighborInfo.link.NodeX().NodeId(), neighborInfo.link.NodeY().NodeId())
		fmt.Printf("追加情報 neighbors：")
		fmt.Println(neighborInfo.info.Neighbors)
	}
}
