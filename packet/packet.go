package packet

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Header struct {
	SourceMac      string
	DestinationMac string
}

type Packet struct {
	Header       Header
	Payload      string
	Size         float64
	Id           string
	creationTime float64
	arrivalTime  float64
}

func NewPacket(s string, d string, header_size float64, payload_size float64, currentTime float64) *Packet {
	p := strings.Repeat("X", int(payload_size))
	size := header_size + payload_size
	return &Packet{
		Header: Header{
			SourceMac:      s,
			DestinationMac: d,
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
