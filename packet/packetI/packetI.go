package packetI

import "nt-simulator/address"

type FragmentFlags struct {
	OriginalDataId string
	MoreFragment   bool
}

type MacHeader struct {
	SourceMac      *address.MacAddress
	DestinationMac *address.MacAddress
}

type IpHeader struct {
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
	GetMacHeader() MacHeader
	GetIpHeader() IpHeader
	SetIpHeader(header IpHeader)
	GetSize() int
	GetId() string
	GetPayload() string
	DecrementTTL()
	GetTTL() int
	RemoveMacHeader()
	AddMacHeader(*address.MacAddress, *address.MacAddress)
}
