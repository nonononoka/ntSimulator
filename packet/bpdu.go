package packet

import (
	"encoding/json"
	"fmt"
	"nt-simulator/address"
)

// Packetをembedding．こうすることで自動的にPacketI interfaceを満たす
type BPDU struct {
	Packet
}

type bpduPayload struct {
	RootID   int     `json:"rootID"`
	BridgeID int     `json:"bridgeID"`
	PathCost float64 `json:"pathCost"`
}

func NewBPDU(s *address.MacAddress, d *address.MacAddress, ttl int, currentTime float64, rootID int, bridgeID int, pathCost float64) *BPDU {
	b, err := json.Marshal(bpduPayload{RootID: rootID, BridgeID: bridgeID, PathCost: pathCost})
	if err != nil {
		panic(fmt.Sprintf("BPDU payload marshal error: %v", err))
	}
	payload := string(b)

	return &BPDU{ // BPDUは，データリンク層の話だからIPアドレスは使わないのでnil
		Packet: *NewPacket(s, d, nil, nil, ttl, 20, len(payload), currentTime, payload),
	}
}

func (b *BPDU) ParsePayload() (bpduPayload, error) {
	var bp bpduPayload
	err := json.Unmarshal([]byte(b.Payload), &bp)
	return bp, err
}

