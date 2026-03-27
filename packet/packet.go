package packet

type Packet struct {
	Source      string
	Destination string
	Payload     string
}

func NewPacket(s string, d string, p string) *Packet {
	return &Packet{Source: s, Destination: d, Payload: p}
}
