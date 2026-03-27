package main

import (
	"nt-simulator/networkgraph"
	"nt-simulator/node"
	"nt-simulator/packet"
)

func main() {
	ng := networkgraph.NewNetworkGraph()
	n1 := node.NewNode(1, "00:01", ng)
	n2 := node.NewNode(2, "00:02", ng)
	l := node.NewLink(n1, n2, 10000, 0.001, 0.0, ng)
	n1.PrintNode()
	n2.PrintNode()
	l.PrintLink()

	// packet転送のテスト
	p := packet.NewPacket(n1.Address(), n2.Address(), "Hello World!")
	n1.SendPacket(p)
	ng.Visualize()
}
