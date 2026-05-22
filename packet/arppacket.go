package packet

import (
	"encoding/json"
	"fmt"
	"nt-simulator/address"
)

type ArpPayload struct {
	Operation string `json:"operation"`
	SourceMac string `json:"sourceMac"`
	DestMac   string `json:"destMac"`
	SourceIp  string `json:"sourceIp"`
	DestIp    string `json:"destIp"`
}

type ArpP struct {
	Packet
}

func NewArpP(s *address.MacAddress, d *address.MacAddress, sourceip *address.IpAddress, destip *address.IpAddress, currentTime float64, operation string) *ArpP {
	p, err := json.Marshal(ArpPayload{Operation: operation, SourceMac: s.String(), DestMac: d.String(), SourceIp: sourceip.String(), DestIp: destip.String()})
	if err != nil {
		panic(fmt.Sprintf("Hello payload marshal error: %v", err))
	}
	payload := string(p)

	return &ArpP{
		Packet: *NewPacket(s, d, sourceip, destip, 1, 28, 28, currentTime, payload),
	}
}
