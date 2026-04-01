package nteventsched

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"time"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
)

type NtEventSched struct {
	currentTime time.Time
	eventId     int
	logEnabled  bool
	verbose     bool
	*NetworkGraph
}

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

func NewNtEventSched() *NtEventSched {
	return &NtEventSched{
		NetworkGraph: newNetworkGraph(),
	}
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
