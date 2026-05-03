package network

import (
	"fmt"
	"math"
	"nt-simulator/nteventsched"
	"nt-simulator/packet"
)

type Switch struct {
	nes            *nteventsched.NtEventSched
	nodeId         int
	links          []*Link
	forwadingTable map[string]*Link
	linkStates     map[*Link]string // Linkの状態を管理するmap
	rootId         int              // 初期状態では自身をルートとする
	rootPathCost   float64          // rootまでのパスコスト
	isRoot         bool
}

func NewSwitch(nes *nteventsched.NtEventSched, nodeId int) *Switch {
	s := &Switch{
		nes:            nes,
		nodeId:         nodeId,
		forwadingTable: make(map[string]*Link),
		linkStates:     make(map[*Link]string),
		rootId:         nodeId,
		rootPathCost:   0,
		isRoot:         true,
	}
	nes.AddNode(s)
	return s
}

func (s *Switch) NodeId() int {
	return s.nodeId
}

func (s *Switch) NodeColor() string { return "red" }

// スイッチに新しいリンクを追加
// BPDUを送信
func (s *Switch) AddLink(link *Link) {
	for _, l := range s.links {
		if l == link {
			return
		}
	}
	s.links = append(s.links, link)
	s.linkStates[link] = "initial"
	s.sendBPDU()

}

func (s *Switch) sendBPDU() {
	for _, l := range s.links {
		// ブロードキャストアドレス：FF:FF:FF:FF:FF:FF
		bpdu := packet.NewBPDU("00:00:00:00:00:00", "FF:FF:FF:FF:FF:FF", s.nes.CurrentTime, s.rootId, s.nodeId, s.rootPathCost)
		l.enqueuePacket(bpdu, s)
	}
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
func (s *Switch) receivePacket(p packet.PacketI, l *Link) {
	if bpdu, ok := p.(*packet.BPDU); ok {
		s.nes.LogPacketInfo(bpdu, "BPDU received", s.nodeId)
		s.processBPDU(bpdu, l)
	} else {
		if p.ArrivalTime() == -1 {
			s.nes.LogPacketInfo(p, "lost", s.nodeId)
			return
		}
		s.nes.LogPacketInfo(p, "received", s.nodeId)
		sourceMacAddress := p.GetHeader().SourceMac
		s.UpdateForwardingTable(sourceMacAddress, l)
		s.forwardPacket(p, l)
	}
}

func (s *Switch) processBPDU(bpdu *packet.BPDU, receivedLink *Link) {
	bp, err := bpdu.ParsePayload()
	if err != nil {
		fmt.Printf("BPDU parse error: %v\n", err)
		return
	}
	receivedRootID := bp.RootID
	receivedPathCost := bp.PathCost + receivedLink.getLinkCost()

	fmt.Printf("current time: %v, nodeId: %v processing BPDU receivedRootId: %v, currentRootId: %v, receivedRootPathCost: %v, currentRootPathCost: %v\n", s.nes.CurrentTime, s.nodeId, receivedRootID, s.rootId, receivedPathCost, s.rootPathCost)

	rootInfoChanged := false
	if receivedRootID < s.rootId || (receivedRootID == s.rootId && receivedPathCost < s.rootPathCost) {
		s.rootId = receivedRootID
		s.rootPathCost = receivedPathCost
		s.isRoot = false
		rootInfoChanged = true
	}

	s.updateLinkStates(receivedLink, receivedPathCost)

	// ルート情報が変更されたらBPDUを再送信
	if rootInfoChanged {
		s.sendBPDU()
	}
}

func (s *Switch) updateLinkStates(receivedLink *Link, receivedBPDUPathCost float64) {
	if s.isRoot {
		// ルートブリッジの場合，全てのポートをフォワーディング状態に設定
		for _, link := range s.links {
			s.linkStates[link] = "forwarding"
		}
	} else {
		// 非ルートブリッジの場合，最小コストのリンクを選択してフォワーディングに設定
		bestPathCost := math.MaxFloat64
		var bestLink *Link
		bestLinkId := 0
		for _, l := range s.links {
			if l.isLinkBetweenSwitches() {
				linkPathCost := l.getLinkCost() + receivedBPDUPathCost
				linkId := min(l.node_x.NodeId(), l.node_y.NodeId())
				if linkPathCost < bestPathCost ||
					(linkPathCost == bestPathCost && linkId < bestLinkId) {
					bestPathCost = linkPathCost
					bestLink = l
					bestLinkId = linkId
				}
			}
		}

		for _, l := range s.links {
			if l == bestLink || !l.isLinkBetweenSwitches() {
				s.linkStates[l] = "forwarding"
				s.nes.UpdateEdgeStyle(l.node_x.NodeId(), l.node_y.NodeId(), "")
			} else {
				s.linkStates[l] = "blocking"
				s.nes.UpdateEdgeStyle(l.node_x.NodeId(), l.node_y.NodeId(), "dashed")
			}
		}

		s.PrintLinkStates()
	}
}

func (s *Switch) forwardPacket(p packet.PacketI, receivedLink *Link) {
	destinationAddress := p.GetHeader().DestinationMac
	if l, ok := s.forwadingTable[destinationAddress]; ok {
		if s.linkStates[l] == "forwarding" {
			s.nes.LogPacketInfo(p, "forwarded", s.nodeId)
			l.enqueuePacket(p, s)
		}
	} else {
		// 宛先が不明の場合，ブロードキャスト
		for _, link := range s.links {
			if link != receivedLink {
				if s.linkStates[link] == "forwarding" {
					s.nes.LogPacketInfo(p, "broadcast", s.nodeId)
					link.enqueuePacket(p, s)
				}
			}
		}
	}
}

func (s *Switch) PrintLinkStates() {
	for l, state := range s.linkStates {
		fmt.Printf("Link %v <-> %v: %s\n", l.node_x.NodeId(), l.node_y.NodeId(), state)
	}
}
