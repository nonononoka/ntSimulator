package nswitch

import (
	"fmt"
	"math"
	"nt-simulator/link"
	"nt-simulator/nteventsched"
	"nt-simulator/packet"
)

type Switch struct {
	*link.BaseNode
	forwardingTable map[string]*link.Link
	linkStates      map[*link.Link]string // Linkの状態を管理するmap
	rootId          int                   // 初期状態では自身をルートとする
	rootPathCost    float64               // rootまでのパスコスト
	isRoot          bool
}

func NewSwitch(nes *nteventsched.NtEventSched, nodeId int, ipAddress string, macAddress string) *Switch {
	s := &Switch{
		BaseNode:        link.NewBaseNode(nodeId, nes, macAddress, ipAddress),
		forwardingTable: make(map[string]*link.Link),
		linkStates:      make(map[*link.Link]string),
		rootId:          nodeId,
		rootPathCost:    0,
		isRoot:          true,
	}
	nes.AddNode(s)
	return s
}

func (s *Switch) NodeColor() string { return "red" }

// スイッチに新しいリンクを追加
// BPDUを送信
func (s *Switch) AddLink(link *link.Link, ip string) {
	for _, l := range s.GetLinks() {
		if l == link {
			return
		}
	}
	s.SetLinks(append(s.GetLinks(), link))
	s.linkStates[link] = "initial"
	s.sendBPDU()
}

// スイッチがパケットを受信したとき
func (s *Switch) ReceivePacket(p packet.PacketI, l *link.Link) {
	if bpdu, ok := p.(*packet.BPDU); ok {
		s.GetNES().LogPacketInfo(bpdu, "BPDU received", s.NodeId())
		s.processBPDU(bpdu, l)
	} else {
		if p.ArrivalTime() == -1 {
			s.GetNES().LogPacketInfo(p, "lost", s.NodeId())
			return
		}
		s.GetNES().LogPacketInfo(p, "received", s.NodeId())
		sourceMacAddress := p.GetHeader().SourceMac
		s.updateForwardingTable(sourceMacAddress, l)
		s.forwardPacket(p, l)
	}
}

func (s *Switch) PrintNode() {
	switchInfo := ""
	for _, l := range s.GetLinks() {
		switchInfo += fmt.Sprintf("%v <-> %v, ", l.NodeX().NodeId(), l.NodeY().NodeId())
	}
	fmt.Printf("スイッチ ノード(ID: %v\n", s.NodeId())
}

func (s *Switch) PrintForwadingTable() {
	for macAddress, l := range s.forwardingTable {
		fmt.Printf("MACアドレス: %s, リンク先ノード %v <-> %v\n", macAddress, l.NodeX().NodeId(), l.NodeY().NodeId())
	}
}

func (s *Switch) PrintLinkStates() {
	for l, state := range s.linkStates {
		fmt.Printf("Link %v <-> %v: %s\n", l.NodeX().NodeId(), l.NodeY().NodeId(), state)
	}
}

func (s *Switch) sendBPDU() {
	for _, l := range s.GetLinks() {
		// ブロードキャストアドレス：FF:FF:FF:FF:FF:FF
		bpdu := packet.NewBPDU("00:00:00:00:00:00", "FF:FF:FF:FF:FF:FF", "00:00:00:00", "255:255:255:255", 64, s.GetNES().CurrentTime, s.rootId, s.NodeId(), s.rootPathCost)
		l.EnqueuePacket(bpdu, s)
	}
}

func (s *Switch) updateForwardingTable(destinationAddress string, link *link.Link) {
	// 宛先アドレスとその宛先へのパケットを転送するためのリンク
	s.forwardingTable[destinationAddress] = link
}

func (s *Switch) processBPDU(bpdu *packet.BPDU, receivedLink *link.Link) {
	bp, err := bpdu.ParsePayload()
	if err != nil {
		fmt.Printf("BPDU parse error: %v\n", err)
		return
	}
	receivedRootID := bp.RootID
	receivedPathCost := bp.PathCost + receivedLink.GetLinkCost()

	fmt.Printf("current time: %v, nodeId: %v processing BPDU receivedRootId: %v, currentRootId: %v, receivedRootPathCost: %v, currentRootPathCost: %v\n", s.GetNES().CurrentTime, s.NodeId(), receivedRootID, s.rootId, receivedPathCost, s.rootPathCost)

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

func (s *Switch) updateLinkStates(receivedLink *link.Link, receivedBPDUPathCost float64) {
	if s.isRoot {
		// ルートブリッジの場合，全てのポートをフォワーディング状態に設定
		for _, link := range s.GetLinks() {
			s.linkStates[link] = "forwarding"
		}
	} else {
		// 非ルートブリッジの場合，最小コストのリンクを選択してフォワーディングに設定
		bestPathCost := math.MaxFloat64
		var bestLink *link.Link
		bestLinkId := 0
		for _, l := range s.GetLinks() {
			if isLinkBetweenSwitches(l) {
				linkPathCost := l.GetLinkCost() + receivedBPDUPathCost
				linkId := min(l.NodeX().NodeId(), l.NodeY().NodeId())
				if linkPathCost < bestPathCost ||
					(linkPathCost == bestPathCost && linkId < bestLinkId) {
					bestPathCost = linkPathCost
					bestLink = l
					bestLinkId = linkId
				}
			}
		}

		for _, l := range s.GetLinks() {
			if l == bestLink || !isLinkBetweenSwitches(l) {
				s.linkStates[l] = "forwarding"
				s.GetNES().UpdateEdgeStyle(l.NodeX().NodeId(), l.NodeY().NodeId(), "")
			} else {
				s.linkStates[l] = "blocking"
				s.GetNES().UpdateEdgeStyle(l.NodeX().NodeId(), l.NodeY().NodeId(), "dashed")
			}
		}

		s.PrintLinkStates()
	}
}

func (s *Switch) forwardPacket(p packet.PacketI, receivedLink *link.Link) {
	destinationAddress := p.GetHeader().DestinationMac
	if l, ok := s.forwardingTable[destinationAddress]; ok {
		if s.linkStates[l] == "forwarding" {
			s.GetNES().LogPacketInfo(p, "forwarded", s.NodeId())
			l.EnqueuePacket(p, s)
		}
	} else {
		// 宛先が不明の場合，ブロードキャスト
		for _, link := range s.GetLinks() {
			if link != receivedLink {
				if s.linkStates[link] == "forwarding" {
					s.GetNES().LogPacketInfo(p, "broadcast", s.NodeId())
					link.EnqueuePacket(p, s)
				}
			}
		}
	}
}

func isLinkBetweenSwitches(l *link.Link) bool {
	_, okX := l.NodeX().(*Switch)
	_, okY := l.NodeY().(*Switch)
	return okX && okY
}
