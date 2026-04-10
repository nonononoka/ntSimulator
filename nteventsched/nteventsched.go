package nteventsched

import (
	"container/heap"
	"fmt"
	"math"
	"os"
	"os/exec"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
)

// priority queueに突っ込むnetwork eventの型
type Event struct {
	eventTime int
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
	CurrentTime int
	eventId     int
	logEnabled  bool
	verbose     bool
	*NetworkGraph
}

func (nes *NtEventSched) Run() {
	pq := nes.events
	for pq.Len() > 0 {
		event := heap.Pop(&pq).(*Event)
		eventTime := event.eventTime
		callback := event.callback
		args := event.args
		callback(args...)
		nes.CurrentTime = eventTime
	}
}

func (nes *NtEventSched) ScheduleEvent(eventTime int, callback func(args ...any), args ...any) {
	heap.Push(&nes.events, &Event{eventTime: eventTime, eventId: nes.eventId, callback: callback, args: args})
	nes.eventId += 1
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

// heap.Initは既存の要素をヒープ順に並べ直すためのもので，空スライスならそのままで動く
func NewNtEventSched() *NtEventSched {
	sched := &NtEventSched{
		NetworkGraph: newNetworkGraph(),
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
