package basenode

import (
	"nt-simulator/link"
	"nt-simulator/nteventsched"
)

type BaseNode struct {
	nodeId int
	links  []*link.Link
	nes    *nteventsched.NtEventSched
}

func NewBaseNode(nodeId int, nes *nteventsched.NtEventSched) *BaseNode {
	return &BaseNode{nodeId: nodeId, nes: nes}
}

func (b *BaseNode) GetLinks() []*link.Link {
	return b.links
}

func (b *BaseNode) SetLinks(links []*link.Link) {
	b.links = links
}

func (b *BaseNode) NodeId() int {
	return b.nodeId
}

func (b *BaseNode) GetNES() *nteventsched.NtEventSched {
	return b.nes
}
