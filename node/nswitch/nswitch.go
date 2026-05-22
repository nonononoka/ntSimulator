package nswitch

import (
	"fmt"
	"math"
	"nt-simulator/address"
	"nt-simulator/link"
	"nt-simulator/node/basenode"
	"nt-simulator/nteventsched"
	"nt-simulator/packet"
	"nt-simulator/packet/packetI"
)

type nswitch struct {
	*basenode.BaseNode
	forwardingTable map[*address.MacAddress]*link.Link
	linkStates      map[*link.Link]string // Linkの状態を管理するmap
	rootId          int                   // 初期状態では自身をルートとする
	rootPathCost    float64               // rootまでのパスコスト
	isRoot          bool
	*address.IpAddress
}

func NewSwitch(nes *nteventsched.NtEventSched, nodeId int, ipAddress string) *nswitch {
	s := &nswitch{
		BaseNode:        basenode.NewBaseNode(nodeId, nes),
		forwardingTable: make(map[*address.MacAddress]*link.Link),
		linkStates:      make(map[*link.Link]string),
		rootId:          nodeId,
		rootPathCost:    0,
		isRoot:          true,
		IpAddress:       address.NewIPAddress(ipAddress),
	}
	nes.AddNode(s)
	return s
}

func (s *nswitch) NodeColor() string { return "red" }

// スイッチに新しいリンクを追加
// BPDUを送信
func (s *nswitch) AddLink(link *link.Link, ip *address.IpAddress) {
	for _, l := range s.GetLinks() {
		if l == link {
			return
		}
	}
	s.SetLinks(append(s.GetLinks(), link))
	// ルートブリッジは即座にフォワーディング状態にする（BPDUを待たない）
	if s.isRoot {
		s.linkStates[link] = "forwarding"
	} else {
		s.linkStates[link] = "initial"
	}
	s.sendBPDU()
}

// スイッチがパケットを受信したとき
func (s *nswitch) ReceivePacket(p packetI.PacketI, l *link.Link) {
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

func (s *nswitch) PrintNode() {
	switchInfo := ""
	for _, l := range s.GetLinks() {
		switchInfo += fmt.Sprintf("%v <-> %v, ", l.NodeX().NodeId(), l.NodeY().NodeId())
	}
	fmt.Printf("スイッチ ノード(ID: %v\n", s.NodeId())
}

func (s *nswitch) PrintForwadingTable() {
	for macAddress, l := range s.forwardingTable {
		fmt.Printf("MACアドレス: %s, リンク先ノード %v <-> %v\n", macAddress, l.NodeX().NodeId(), l.NodeY().NodeId())
	}
}

func (s *nswitch) PrintLinkStates() {
	for l, state := range s.linkStates {
		fmt.Printf("Link %v <-> %v: %s\n", l.NodeX().NodeId(), l.NodeY().NodeId(), state)
	}
}

func (s *nswitch) GetIPAddresses() []*address.IpAddress {
	return []*address.IpAddress{s.IpAddress}
}

func (s *nswitch) GetLinkState(l *link.Link) string {
	return s.linkStates[l]
}

func (s *nswitch) sendBPDU() {
	for _, l := range s.GetLinks() {
		// ブロードキャストアドレス：FF:FF:FF:FF:FF:FF
		bpdu := packet.NewBPDU(address.NewMacAddress("00:00:00:00:00:00"), address.NewMacAddress("FF:FF:FF:FF:FF:FF"), 64, s.GetNES().CurrentTime, s.rootId, s.NodeId(), s.rootPathCost)
		l.EnqueuePacket(bpdu, s)
	}
}

func (s *nswitch) updateForwardingTable(destinationAddress *address.MacAddress, link *link.Link) {
	// 宛先アドレスとその宛先へのパケットを転送するためのリンク
	s.forwardingTable[destinationAddress] = link
}

func (s *nswitch) processBPDU(bpdu *packet.BPDU, receivedLink *link.Link) {
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

func (s *nswitch) updateLinkStates(receivedLink *link.Link, receivedBPDUPathCost float64) {
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

func (s *nswitch) forwardPacket(p packetI.PacketI, receivedLink *link.Link) {
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
	_, okX := l.NodeX().(*nswitch)
	_, okY := l.NodeY().(*nswitch)
	return okX && okY
}
