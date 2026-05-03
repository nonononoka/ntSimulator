package nteventsched

import (
	"fmt"
	"math"
	"os"
	"os/exec"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
)

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

type GraphNode interface {
	NodeId() int
	NodeColor() string
}

func (ng *NetworkGraph) AddNode(n GraphNode) {
	attrs := []func(*graph.VertexProperties){}
	if color := n.NodeColor(); color != "" {
		attrs = append(attrs, graph.VertexAttribute("style", "filled"))
		attrs = append(attrs, graph.VertexAttribute("fillcolor", color))
	}
	ng.G.AddVertex(n.NodeId(), attrs...)
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

func (ng *NetworkGraph) UpdateEdgeStyle(fromNodeId int, toNodeId int, style string) {
	ng.G.UpdateEdge(fromNodeId, toNodeId, graph.EdgeAttribute("style", style))
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
