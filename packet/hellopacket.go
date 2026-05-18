package packet

import (
	"encoding/json"
	"fmt"
	"nt-simulator/address"
)

// OSPFプロトコルで使われるHelloパケット。
// ルーターはまずHelloパケットを送信して周囲のルーターを検出する。
// これでPacketIを自動で満たす
type HelloP struct{
	Packet
}

type HellopPayload struct{
	RouterId int `json:"routerID"`
	HelloInterval float64 `json:"helloInterval"`
	Neighbors []int `json:"neighbors"`
}

func NewHelloP(s *address.MacAddress, sourceip *address.IpAddress, currentTime float64, routerId int, helloInterval float64, neighbors []int) *HelloP{
	p, err := json.Marshal(HellopPayload{ RouterId: routerId, HelloInterval: helloInterval, Neighbors: neighbors})
	if err != nil {
		panic(fmt.Sprintf("BPDU payload marshal error: %v", err))
	}
	payload := string(p)

	return &HelloP{
		Packet: *NewPacket(s, address.NewMacAddress("FF:FF:FF:FF:FF:FF"), sourceip, address.NewIPAddress("224.0.0.5/32"), 1, 24, 20, currentTime, payload),
	}
}

func (h *HelloP) ParsePayload() (HellopPayload, error) {
	var hp HellopPayload
	err := json.Unmarshal([]byte(h.Payload), &hp)
	return hp, err
}

func (hp1 HellopPayload) Equals(hp2 HellopPayload) bool{
	if hp1.RouterId != hp2.RouterId{
		return false
	}
	if hp1.HelloInterval != hp2.HelloInterval{
		return false
	}
	if len(hp1.Neighbors) != len(hp2.Neighbors){
		return false
	}
	counts := make(map[int]int)
	for _, x := range hp1.Neighbors {
		counts[x]++
	}

	// bのスライスの要素で引き算していく
	for _, x := range hp2.Neighbors {
		if counts[x] == 0 {
			return false // aに存在しない、または個数が合わない
		}
		counts[x]--
	}
	return true
}





