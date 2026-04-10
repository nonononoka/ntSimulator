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
	Size         int
	Id           string
	creationTime int
	arrivalTime  int
}

func NewPacket(s string, d string, header_size int, payload_size int, nes *nteventsched.NtEventSched) *Packet {
	p := strings.Repeat("X", payload_size)
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
