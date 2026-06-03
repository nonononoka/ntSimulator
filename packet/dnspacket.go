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

func NewDNSP(s *address.MacAddress, sourceip *address.IpAddress, currentTime float64, routerId int, helloInterval float64, neighbors []int, queryDomain string, queryType string, resolvedIp string) *DNSP {
	p, err := json.Marshal(DNSPayload{QueryDomain: queryDomain, QueryType: queryType, ResolvedIp: resolvedIp})
	if err != nil {
		panic(fmt.Sprintf("Hello payload marshal error: %v", err))
	}
	payload := string(p)
	return &DNSP{
		Packet: *NewPacket(s, address.NewMacAddress("FF:FF:FF:FF:FF:FF"), sourceip, address.NewIPAddress("224.0.0.5/32"), 64, 0, 0, currentTime, payload),
	}
}

func (h *DNSP) ParsePayload() (DNSPayload, error) {
	var hp DNSPayload
	err := json.Unmarshal([]byte(h.Payload), &hp)
	return hp, err
}
