package network

import (
	"fmt"
	"nt-simulator/nteventsched"
	"nt-simulator/packet"
)

type Switch struct {
	nes            *nteventsched.NtEventSched
	nodeId         int
	links          []*Link
	forwadingTable map[string]*Link
}

func NewSwitch(nes *nteventsched.NtEventSched, nodeId int) *Switch {
	s := &Switch{
		nes:            nes,
		nodeId:         nodeId,
		forwadingTable: make(map[string]*Link),
	}
	nes.AddNode(s)
	return s
}

func (s *Switch) NodeId() int {
	return s.nodeId
}

func (s *Switch) NodeColor() string { return "red" }

// スイッチに新しいリンクを追加
func (s *Switch) AddLink(link *Link) {
	for _, l := range s.links {
		if l == link {
			return
		}
	}
	s.links = append(s.links, link)
}

func (s *Switch) PrintNode() {
	switchInfo := ""
	for _, l := range s.links {
		switchInfo += fmt.Sprintf("%v <-> %v, ", l.node_x.NodeId(), l.node_y.NodeId())
	}
	fmt.Printf("スイッチ ノード(ID: %v\n", s.nodeId)
}

func (s *Switch) UpdateForwardingTable(destionationAddress string, link *Link) {
	// 宛先アドレスとその宛先へのパケットを転送するためのリンク
	s.forwadingTable[destionationAddress] = link
}

// スイッチがパケットを受信したとき
func (s *Switch) receivePacket(p *packet.Packet) {
	if p.ArrivalTime() == -1 {
		s.nes.LogPacketInfo(p, "lost", s.nodeId)
		return
	}
	s.nes.LogPacketInfo(p, "received", s.nodeId)
	s.forwardPacket(p)
}

func (s *Switch) forwardPacket(p *packet.Packet) {
	destinationAddress := p.Header.Destination
	if l, ok := s.forwadingTable[destinationAddress]; ok {
		s.nes.LogPacketInfo(p, "forwarded", s.nodeId)
		l.enqueuePacket(p, s)
	}
	// 宛先がテーブルにない場合の処理は未実装
}
