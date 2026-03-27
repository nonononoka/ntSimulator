package node

import (
	"fmt"
	"nt-simulator/networkgraph"
	"nt-simulator/packet"
)

type N struct {
	nodeId       int
	address      string
	links        []Link
	networkgraph *networkgraph.NetworkGraph
}

type Link struct {
	node_x       *N
	node_y       *N
	bandwidth    float64
	delay        float64
	packet_loss  float64
	networkgraph *networkgraph.NetworkGraph
}

func (n *N) Address() string {
	return n.address
}

func NewNode(node_id int, address string, networkgraph *networkgraph.NetworkGraph) *N {
	networkgraph.AddNode(node_id)
	return &N{nodeId: node_id, address: address, networkgraph: networkgraph}
}

func NewLink(node_x *N, node_y *N, bandwidth float64, delay float64, packet_loss float64, networkgraph *networkgraph.NetworkGraph) *Link {
	networkgraph.AddEdge(node_x.nodeId, node_y.nodeId, fmt.Sprintf("%v Mbps %v s\n", bandwidth/1000000, delay), bandwidth, delay)
	l := Link{node_x: node_x, node_y: node_y, bandwidth: bandwidth, delay: delay, packet_loss: packet_loss, networkgraph: networkgraph}
	node_x.addLink(l)
	node_y.addLink(l)
	return &l
}

func (n *N) PrintNode() {
	connected_nodes := make([]int, 0, 10)
	for _, v := range n.links {
		if v.node_x.nodeId != n.nodeId {
			connected_nodes = append(connected_nodes, v.node_x.nodeId)
		}
		if v.node_y.nodeId != n.nodeId {
			connected_nodes = append(connected_nodes, v.node_y.nodeId)
		}
	}
	fmt.Printf("ノード(ID: %v, アドレス: %s), 接続ノード: %v\n", n.nodeId, n.address, connected_nodes)
}

func (l *Link) PrintLink() {
	fmt.Printf("%v <-> %v 帯域幅: %v 遅延: %v パケットロス率：%v\n", l.node_x.nodeId, l.node_y.nodeId, l.bandwidth, l.delay, l.packet_loss)
}

func (l *Link) transferPacket(p *packet.Packet, from_node *N) {
	var toNode *N
	if l.node_x.nodeId != from_node.nodeId {
		toNode = l.node_x
	} else {
		toNode = l.node_y
	}
	toNode.receivePacket(p)
}

func (n *N) addLink(link Link) {
	// TODO:linkがすでにlinksにないことを確認しなきゃいけない．
	n.links = append(n.links, link)
}

func (n *N) SendPacket(p *packet.Packet) {
	if p.Destination == n.address {
		n.receivePacket(p)
	} else {
		for _, l := range n.links {
			var nextNodeId int
			var from_node *N = n
			if l.node_x.nodeId != n.nodeId {
				nextNodeId = l.node_x.nodeId
			} else {
				nextNodeId = l.node_y.nodeId
			}
			fmt.Printf("ノード%vからノード%vにパケットを転送\n", n.nodeId, nextNodeId)
			l.transferPacket(p, from_node)
		}
	}
}

func (n *N) receivePacket(p *packet.Packet) {
	fmt.Printf("ノード%vがパケットを受信: %s\n", n.nodeId, p.Payload)
}
