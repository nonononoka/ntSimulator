package nteventsched

import (
	"fmt"
	"math"
	"nt-simulator/address"
	"nt-simulator/packet"
)

type packetEvent struct {
	time     float64
	event    string
	nodeId   int
	packetId string
	src      *address.MacAddress
	dst      *address.MacAddress
}

type packetLog struct {
	source         *address.MacAddress
	destination    *address.MacAddress
	size           int
	creationTime   float64
	arrivalTime    float64
	originalDataId string
	events         []*packetEvent
}

func (nes *NtEventSched) LogPacketInfo(p packet.PacketI, eventType string, nodeId int) {
	if !nes.logEnabled {
		return
	}
	_, ok := nes.packetLogs[p.GetId()]
	if !ok {
		nes.packetLogs[p.GetId()] = &packetLog{
			source:         p.GetHeader().SourceMac,
			destination:    p.GetHeader().DestinationMac,
			size:           p.GetSize(),
			creationTime:   p.CreationTime(),
			arrivalTime:    p.ArrivalTime(),
			originalDataId: p.GetHeader().FragmentFlags.OriginalDataId,
		}
	}

	switch eventType {
	case "arrived":
		nes.packetLogs[p.GetId()].arrivalTime = nes.CurrentTime
	case "lost":
		nes.packetLogs[p.GetId()].arrivalTime = -1
	}

	eventInfo := packetEvent{
		time:     nes.CurrentTime,
		event:    eventType,
		nodeId:   nodeId,
		packetId: p.GetId(),
		src:      p.GetHeader().SourceMac,
		dst:      p.GetHeader().DestinationMac,
	}
	nes.packetLogs[p.GetId()].events = append(nes.packetLogs[p.GetId()].events, &eventInfo)

	if nes.Verbose {
		fmt.Printf("time: %v, node: %v, event: %s, packet: %v, src: %s, dst: %s\n",
			nes.CurrentTime, nodeId, eventType, p.GetId(), p.GetHeader().SourceMac, p.GetHeader().DestinationMac)
	}
}

func (nes *NtEventSched) PrintPacketLogs() {
	for packetId, log := range nes.packetLogs {
		fmt.Printf("Packet ID: %v, Src: %s %v -> Dst: %s %v\n",
			packetId, log.source, log.creationTime, log.destination, log.arrivalTime)
		for _, event := range log.events {
			fmt.Printf("time: %v, event: %s nodeId: %v\n", event.time, event.event, event.nodeId)
		}
	}
}

func (nes *NtEventSched) GenerateSummary() {
	type flowStats struct {
		sentPackets   int
		sentBytes     float64
		recvPackets   int
		recvBytes     float64
		lostPackets   int
		totalDelay    float64
		firstCreation float64
		lastArrival   float64
	}
	type fragGroup struct {
		flowKey       string
		totalBytes    float64
		firstCreation float64
		lastArrival   float64
		hasSent       bool
		isReceived    bool
	}

	flows := make(map[string]*flowStats)
	fragGroups := make(map[string]*fragGroup)

	initFlow := func(key string) {
		if _, ok := flows[key]; !ok {
			flows[key] = &flowStats{firstCreation: math.MaxFloat64}
		}
	}

	for _, log := range nes.packetLogs {
		key := log.source.String() + " -> " + log.destination.String()

		if log.originalDataId != "" {
			// フラグメント: originalDataId でグループ化して1論理パケットとして扱う
			if _, ok := fragGroups[log.originalDataId]; !ok {
				fragGroups[log.originalDataId] = &fragGroup{
					flowKey:       key,
					firstCreation: math.MaxFloat64,
				}
			}
			fg := fragGroups[log.originalDataId]
			for _, e := range log.events {
				if e.event == "sent" {
					fg.hasSent = true
					fg.totalBytes += float64(log.size)
				}
				if e.event == "reassembled" || e.event == "processed" {
					fg.isReceived = true
				}
			}
			if log.creationTime < fg.firstCreation {
				fg.firstCreation = log.creationTime
			}
			if log.arrivalTime > 0 && log.arrivalTime > fg.lastArrival {
				fg.lastArrival = log.arrivalTime
			}
		}
		// originalDataId == "" は断片化前の元パケット ("created" イベントのみ) なので無視
	}

	for _, fg := range fragGroups {
		if !fg.hasSent {
			continue
		}
		initFlow(fg.flowKey)
		f := flows[fg.flowKey]
		f.sentPackets++
		f.sentBytes += fg.totalBytes
		if fg.firstCreation < f.firstCreation {
			f.firstCreation = fg.firstCreation
		}
		if fg.isReceived {
			f.recvPackets++
			f.recvBytes += fg.totalBytes
			f.totalDelay += fg.lastArrival - fg.firstCreation
			if fg.lastArrival > f.lastArrival {
				f.lastArrival = fg.lastArrival
			}
		} else {
			f.lostPackets++
		}
	}

	for key, f := range flows {
		var avgDelay float64
		if f.recvPackets > 0 {
			avgDelay = f.totalDelay / float64(f.recvPackets)
		}

		var throughput float64
		if duration := f.lastArrival - f.firstCreation; duration > 0 {
			throughput = f.recvBytes * 8 / duration
		}

		fmt.Printf("=== Flow: %s ===\n", key)
		fmt.Printf("  Sent:           %d packets, %.0f bytes\n", f.sentPackets, f.sentBytes)
		fmt.Printf("  Received:       %d packets, %.0f bytes\n", f.recvPackets, f.recvBytes)
		fmt.Printf("  Lost:           %d packets\n", f.lostPackets)
		fmt.Printf("  Avg Throughput: %.2f bps\n", throughput)
		fmt.Printf("  Avg Delay:      %.6f s\n", avgDelay)
	}
}
