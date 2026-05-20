package packetI

import "nt-simulator/address"

type FragmentFlags struct {
	OriginalDataId string
	MoreFragment   bool
}

type Header struct {
	SourceMac      *address.MacAddress
	DestinationMac *address.MacAddress
	SourceIp       *address.IpAddress
	DestIp         *address.IpAddress
	TTL            int
	FragmentFlags  FragmentFlags
	FragmentOffset int
}

type PacketI interface {
	SetArrived(time float64)
	ArrivalTime() float64
	CreationTime() float64
	PrintPacket()
	GetHeader() Header
	GetSize() int
	GetId() string
	GetPayload() string
	DecrementTTL()
	GetTTL() int
}
