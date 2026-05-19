package packet

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
