package nteventsched

import (
	"container/heap"
	"fmt"
	"math"
	"nt-simulator/packet"
	"os"
	"os/exec"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
)

// priority queueに突っ込むnetwork eventの型
type Event struct {
	eventTime float64
	eventId   int
	args      []any
	callback  func(args ...any)
}

type PriorityQueue []*Event

// priorityQueueがheapをimplementするためのメソッドたち

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].eventTime < pq[j].eventTime
}

func (pq PriorityQueue) Swap(i, j int) { pq[i], pq[j] = pq[j], pq[i] }

func (pq *PriorityQueue) Push(x any) {
	*pq = append(*pq, x.(*Event)) // type assertion
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[:n-1]
	return item
}

type NtEventSched struct {
	events      PriorityQueue
	CurrentTime float64
	eventId     int
	logEnabled  bool
	verbose     bool
	packetLogs  map[string]*packetLog
	*NetworkGraph
}

func (nes *NtEventSched) Run() {
	for nes.events.Len() > 0 {
		event := heap.Pop(&nes.events).(*Event)
		nes.CurrentTime = event.eventTime
		event.callback(event.args...)
	}
}

func (nes *NtEventSched) ScheduleEvent(eventTime float64, callback func(args ...any), args ...any) {
	heap.Push(&nes.events, &Event{eventTime: eventTime, eventId: nes.eventId, callback: callback, args: args})
	nes.eventId += 1
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

// network graph関連
type NetworkGraph struct {
	G graph.Graph[int, int]
}

func newNetworkGraph() *NetworkGraph {
	return &NetworkGraph{
		G: graph.New(
			graph.IntHash,
		),
	}
}

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

// heap.Initは既存の要素をヒープ順に並べ直すためのもので，nilスライスならそのままで動く．
func NewNtEventSched(logEnabled bool, verbose bool) *NtEventSched {
	sched := &NtEventSched{
		NetworkGraph: newNetworkGraph(),
		logEnabled:   logEnabled,
		verbose:      verbose,
		packetLogs:   make(map[string]*packetLog),
	}
	return sched
}

func (ng *NetworkGraph) AddNode(nodeId int) {
	ng.G.AddVertex(nodeId)
}

func getEdgeWidth(bandwidth float64) float64 {
	return math.Log10(bandwidth)
}

func getEdgeColor(delay float64) string {
	if delay <= 0.001 {
		return "green"
	} else if delay <= 0.01 {
		return "yellow"
	} else {
		return "red"
	}
}

func (ng *NetworkGraph) AddEdge(fromNodeId int, toNodeId int, edgeLabel string, bandwidth float64, delay float64) {
	ng.G.AddEdge(fromNodeId, toNodeId, graph.EdgeAttribute("label", edgeLabel), graph.EdgeAttribute("color", getEdgeColor(delay)), graph.EdgeAttribute(
		"penwidth",
		fmt.Sprintf("%.2f", getEdgeWidth(bandwidth)),
	))
}

func (ng NetworkGraph) Visualize() error {
	file, err := os.Create("network.gv")
	if err != nil {
		return err
	}
	defer file.Close()

	if err := draw.DOT(ng.G, file); err != nil {
		return err
	}

	// dotコマンド実行
	cmd := exec.Command(
		"dot",
		"-Tsvg",
		"network.gv",
		"-o",
		"network.svg",
	)

	return cmd.Run()
}
