package packet

import (
	"fmt"
	"nt-simulator/address"
	"nt-simulator/packet/packetI"

	"github.com/google/uuid"
)

// normalなパケット

type FragmentFlags struct {
	OriginalDataId string
	MoreFragment   bool
}

type Packet struct {
	Header       packetI.Header
	Payload      string
	Size         int
	Id           string
	creationTime float64
	arrivalTime  float64
}

func NewFragment(s *address.MacAddress, d *address.MacAddress, sourceip *address.IpAddress, destip *address.IpAddress, ttl int, header_size int, currentTime float64, originalDataId string, morefragment bool, offset int, p string) *Packet {
	size := header_size + len(p)
	return &Packet{
		Header: packetI.Header{
			SourceMac:      s,
			DestinationMac: d,
			SourceIp:       sourceip,
			DestIp:         destip,
			TTL:            ttl,
			FragmentFlags:  packetI.FragmentFlags{OriginalDataId: originalDataId, MoreFragment: morefragment},
			FragmentOffset: offset,
		},
		Payload:      p,
		Size:         size,
		Id:           uuid.New().String(),
		creationTime: currentTime,
	}
}

func NewPacket(s *address.MacAddress, d *address.MacAddress, sourceip *address.IpAddress, destip *address.IpAddress, ttl int, header_size int, payload_size int, currentTime float64, p string) *Packet {
	size := header_size + payload_size
	return &Packet{
		Header: packetI.Header{
			SourceMac:      s,
			DestinationMac: d,
			SourceIp:       sourceip,
			DestIp:         destip,
		},
		Payload:      p,
		Size:         size,
		Id:           uuid.New().String(),
		creationTime: currentTime,
	}
}

func (p *Packet) PrintPacket() {
	fmt.Printf("パケット(送信元: %s), (宛先:%s), ペイロード: %s", p.Header.SourceMac, p.Header.DestinationMac, p.Payload)
}

func (p *Packet) SetArrived(time float64) {
	p.arrivalTime = time
}

func (p *Packet) ArrivalTime() float64 {
	return p.arrivalTime
}

func (p *Packet) CreationTime() float64 {
	return p.creationTime
}

func (p *Packet) GetHeader() packetI.Header { return p.Header }
func (p *Packet) GetSize() int              { return p.Size }
func (p *Packet) GetId() string             { return p.Id }
func (p *Packet) GetPayload() string        { return p.Payload }
func (p *Packet) DecrementTTL()             { p.Header.TTL = p.Header.TTL - 1 }
func (p *Packet) GetTTL() int               { return p.Header.TTL }
