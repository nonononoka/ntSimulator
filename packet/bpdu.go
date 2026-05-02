package packet

import (
	"encoding/json"
	"fmt"
)

type BPDUPayload struct {
	RootID   int `json:"rootID"`
	BridgeID int `json:"bridgeID"`
	PathCost int `json:"pathCost"`
}

// Packetをembedding．こうすることで自動的にPacketI interfaceを満たす
type BPDU struct {
	Packet
}

func NewBPDU(s string, d string, currentTime float64, rootID int, bridgeID int, pathCost int) *BPDU {
	b, err := json.Marshal(BPDUPayload{RootID: rootID, BridgeID: bridgeID, PathCost: pathCost})
	if err != nil {
		panic(fmt.Sprintf("BPDU payload marshal error: %v", err))
	}
	payload := string(b)

	return &BPDU{
		Packet: *NewPacketWithPayload(s, d, 20, float64(len(payload)), currentTime, payload),
	}
}
