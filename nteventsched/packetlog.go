package nteventsched

import (
	"fmt"
	"math"
	"nt-simulator/packet"
)

type packetEvent struct {
	time     float64
	event    string
	nodeId   int
	packetId string
	src      string
	dst      string
}

type packetLog struct {
	source       string
	destination  string
	size         float64
	creationTime float64
	arrivalTime  float64
	events       []*packetEvent
}

func (nes *NtEventSched) LogPacketInfo(p *packet.Packet, eventType string, nodeId int) {
	if !nes.logEnabled {
		return
	}
	_, ok := nes.packetLogs[p.Id]
	if !ok {
		nes.packetLogs[p.Id] = &packetLog{
			source:       p.Header.Source,
			destination:  p.Header.Destination,
			size:         p.Size,
			creationTime: p.CreationTime(),
			arrivalTime:  p.ArrivalTime(),
		}
	}

	if eventType == "arrived" {
		nes.packetLogs[p.Id].arrivalTime = nes.CurrentTime
	} else if eventType == "lost" {
		nes.packetLogs[p.Id].arrivalTime = -1
	}

	eventInfo := packetEvent{
		time:     nes.CurrentTime,
		event:    eventType,
		nodeId:   nodeId,
		packetId: p.Id,
		src:      p.Header.Source,
		dst:      p.Header.Destination,
	}
	nes.packetLogs[p.Id].events = append(nes.packetLogs[p.Id].events, &eventInfo)

	if nes.verbose {
		fmt.Printf("time: %v, node: %v, event: %s, packet: %v, src: %s, dst: %s\n",
			nes.CurrentTime, nodeId, eventType, p.Id, p.Header.Source, p.Header.Destination)
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

	flows := make(map[string]*flowStats)

	for _, log := range nes.packetLogs {
		key := log.source + " -> " + log.destination
		if _, ok := flows[key]; !ok {
			flows[key] = &flowStats{firstCreation: math.MaxFloat64}
		}
		f := flows[key]
		f.sentPackets++
		f.sentBytes += log.size

		if log.creationTime < f.firstCreation {
			f.firstCreation = log.creationTime
		}

		if log.arrivalTime > 0 {
			f.recvPackets++
			f.recvBytes += log.size
			f.totalDelay += log.arrivalTime - log.creationTime
			if log.arrivalTime > f.lastArrival {
				f.lastArrival = log.arrivalTime
			}
		} else if log.arrivalTime == -1 {
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
