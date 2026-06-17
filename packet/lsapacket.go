package packet

import (
	"encoding/json"
	"fmt"
	"nt-simulator/address"
)

type LsaP struct {
	Packet
}

type LinkStateInfo struct {
	NodeXId   int     `json:"nodeXId"`
	NodeYId   int     `json:"nodeYId"`
	IpAddress string  `json:"ipAddress"`
	Cost      float64 `json:"cost"`
}

type lsaPayload struct {
	RouterId       int             `json:"routerID"`
	SequenceNumber int             `json:"sequenceNumber"`
	LinkStateInfos []LinkStateInfo `json:"stateInfos"`
}

func NewLsaP(s *address.MacAddress, sourceip *address.IpAddress, currentTime float64, routerId int, sequenceNumber int, linkStateInfos []LinkStateInfo) *LsaP {
	p, err := json.Marshal(lsaPayload{RouterId: routerId, SequenceNumber: sequenceNumber, LinkStateInfos: linkStateInfos})
	if err != nil {
		panic(fmt.Sprintf("LSA payload marshal error: %v", err))
	}
	payload := string(p)

	return &LsaP{
		Packet: *NewPacket(s, address.BroadcastMacAddress, sourceip, address.OSPFAllSPFRoutersIPAddress, 1, 24, 100, currentTime, payload),
	}
}

func (l *LsaP) ParsePayload() (lsaPayload, error) {
	var lp lsaPayload
	err := json.Unmarshal([]byte(l.Payload), &lp)
	return lp, err
}
