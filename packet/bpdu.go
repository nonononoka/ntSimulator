package packet

import (
	"encoding/json"
	"fmt"
)

type BPDUPayload struct {
	RootID   int     `json:"rootID"`
	BridgeID int     `json:"bridgeID"`
	PathCost float64 `json:"pathCost"`
}

// Packetをembedding．こうすることで自動的にPacketI interfaceを満たす
type BPDU struct {
	Packet
}

func (b *BPDU) ParsePayload() (BPDUPayload, error) {
	var bp BPDUPayload
	err := json.Unmarshal([]byte(b.Payload), &bp)
	return bp, err
}

func NewBPDU(s string, d string, sourceip string, destip string, ttl int, currentTime float64, rootID int, bridgeID int, pathCost float64) *BPDU {
	b, err := json.Marshal(BPDUPayload{RootID: rootID, BridgeID: bridgeID, PathCost: pathCost})
	if err != nil {
		panic(fmt.Sprintf("BPDU payload marshal error: %v", err))
	}
	payload := string(b)

	return &BPDU{
		Packet: *NewPacket(s, d, sourceip, destip, ttl, 20, len(payload), currentTime, payload),
	}
}
