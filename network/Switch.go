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

func (s *Switch) PrintForwadingTable() {
	for macAddress, l := range s.forwadingTable {
		fmt.Printf("MACアドレス: %s, リンク先ノード %v <-> %v\n", macAddress, l.node_x.NodeId(), l.node_y.NodeId())
	}
}

func (s *Switch) UpdateForwardingTable(destionationAddress string, link *Link) {
	// 宛先アドレスとその宛先へのパケットを転送するためのリンク
	s.forwadingTable[destionationAddress] = link
}

// スイッチがパケットを受信したとき
func (s *Switch) receivePacket(p *packet.Packet, l *Link) {
	if p.ArrivalTime() == -1 {
		s.nes.LogPacketInfo(p, "lost", s.nodeId)
		return
	}
	s.nes.LogPacketInfo(p, "received", s.nodeId)
	sourceMacAddress := p.Header.SourceMac
	s.UpdateForwardingTable(sourceMacAddress, l)
	s.forwardPacket(p, l)
}

func (s *Switch) forwardPacket(p *packet.Packet, receivedLink *Link) {
	destinationAddress := p.Header.DestinationMac
	if l, ok := s.forwadingTable[destinationAddress]; ok {
		s.nes.LogPacketInfo(p, "forwarded", s.nodeId)
		l.enqueuePacket(p, s)
	} else {
		// 宛先が不明の場合，ブロードキャスト
		for _, link := range s.links {
			if link != receivedLink {
				s.nes.LogPacketInfo(p, "broadcast", s.nodeId)
				link.enqueuePacket(p, s)
			}
		}
	}
}
