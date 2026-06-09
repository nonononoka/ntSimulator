package packet

import (
	"encoding/json"
	"fmt"
	"nt-simulator/address"
)

type DHCPP struct {
	Packet
}

type DHCPPayload struct {
	MessageType string `json:"messageType"`
	OfferedIP   string `json:"offeredIP"`
	AssignedIP  string `json:"assignedIP"`
	DnsServerIP string `json:"dnsServerIP"`
	RequestedIP string `json:"requestedIP"`
}

func NewDHCPP(s *address.MacAddress, d *address.MacAddress, sourceip *address.IpAddress, destip *address.IpAddress, currentTime float64, messageType string) *DHCPP {
	p, err := json.Marshal(DHCPPayload{MessageType: messageType})
	if err != nil {
		panic(fmt.Sprintf("Hello payload marshal error: %v", err))
	}
	payload := string(p)
	return &DHCPP{
		Packet: *NewPacket(s, d, sourceip, destip, 64, 0, 0, currentTime, payload),
	}
}

func NewDHCPPWithOfferedIP(s *address.MacAddress, d *address.MacAddress, sourceip *address.IpAddress, destip *address.IpAddress, currentTime float64, messageType string, offeredIP string) *DHCPP {
	p, err := json.Marshal(DHCPPayload{MessageType: messageType, OfferedIP: offeredIP})
	if err != nil {
		panic(fmt.Sprintf("Hello payload marshal error: %v", err))
	}
	payload := string(p)
	return &DHCPP{
		Packet: *NewPacket(s, d, sourceip, destip, 64, 0, 0, currentTime, payload),
	}
}

func NewDHCPPWithAssignedIPAndDNSIP(s *address.MacAddress, d *address.MacAddress, sourceip *address.IpAddress, destip *address.IpAddress, currentTime float64, messageType string, assignedIP string, dnsServerIP string) *DHCPP {
	p, err := json.Marshal(DHCPPayload{MessageType: messageType, AssignedIP: assignedIP, DnsServerIP: dnsServerIP})
	if err != nil {
		panic(fmt.Sprintf("Hello payload marshal error: %v", err))
	}
	payload := string(p)
	return &DHCPP{
		Packet: *NewPacket(s, d, sourceip, destip, 64, 0, 0, currentTime, payload),
	}
}

func (h *DHCPP) ParsePayload() (DHCPPayload, error) {
	var hp DHCPPayload
	err := json.Unmarshal([]byte(h.Payload), &hp)
	return hp, err
}
