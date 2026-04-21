package packet

import (
	"fmt"
	"strings"

	"nt-simulator/nteventsched"

	"github.com/google/uuid"
)

type Header struct {
	Source      string
	Destination string
}

type Packet struct {
	Header       Header
	Payload      string
	Size         float64
	Id           string
	creationTime int
	arrivalTime  int
}

func NewPacket(s string, d string, header_size float64, payload_size float64, nes *nteventsched.NtEventSched) *Packet {
	p := strings.Repeat("X", int(payload_size))
	size := header_size + payload_size
	return &Packet{
		Header: Header{
			Source:      s,
			Destination: d,
		},
		Payload:      p,
		Size:         size,
		Id:           uuid.New().String(),
		creationTime: nes.CurrentTime,
	}
}

func (p *Packet) PrintPacket() {
	fmt.Printf("パケット(送信元: %s), (宛先:%s), ペイロード: %s", p.Header.Source, p.Header.Destination, p.Payload)
}

func (p *Packet) SetArriced(time int) {
	p.arrivalTime = time
}

func (p *Packet) ArrivalTime() int {
	return p.arrivalTime
}
