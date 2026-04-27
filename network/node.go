package network

import "nt-simulator/packet"

// networkに含まれるnode（terminal nodeとかswitchとか含めて）のinterface
type node interface {
	PrintNode()
	NodeId() int
	AddLink(link *Link)
	receivePacket(p *packet.Packet)
}
