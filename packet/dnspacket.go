package packet

import (
	"encoding/json"
	"fmt"
	"nt-simulator/address"
)

type DNSP struct {
	Packet
}

type DNSPayload struct {
	QueryDomain string `json:"queryDomain"`
	QueryType   string `json:"queryType"`
	ResolvedIp  string `json:"resolvedIp"`
}

const DNSQueryTypeA = "A"

func NewDNSP(s *address.MacAddress, d *address.MacAddress, sourceip *address.IpAddress, destip *address.IpAddress, currentTime float64, queryDomain string, queryType string, resolvedIp string) *DNSP {
	p, err := json.Marshal(DNSPayload{QueryDomain: queryDomain, QueryType: queryType, ResolvedIp: resolvedIp})
	if err != nil {
		panic(fmt.Sprintf("Hello payload marshal error: %v", err))
	}
	payload := string(p)
	return &DNSP{
		Packet: *NewPacket(s, d, sourceip, destip, 64, 0, 0, currentTime, payload),
	}
}

func (h *DNSP) ParsePayload() (DNSPayload, error) {
	var hp DNSPayload
	err := json.Unmarshal([]byte(h.Payload), &hp)
	return hp, err
}
