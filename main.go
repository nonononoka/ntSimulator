package main

import (
	"nt-simulator/node"
	"nt-simulator/nteventsched"
)

func main() {
	nes := nteventsched.NewNtEventSched()
	n1 := node.NewNode(1, "00:01", nes)
	n2 := node.NewNode(2, "00:02", nes)
	l1 := node.NewLink(n1, n2, 10000, 0.001, 0.0, nes)
	n1.PrintNode()
	n2.PrintNode()
	l1.PrintLink()

	// // packet転送のテスト（n1→n2）
	// p1 := packet.NewPacket(n1.Address(), n2.Address(), "Hello World from p1!")
	// n1.SendPacket(p1)

	// // packet転送のテスろ（n2→n1）
	// p2 := packet.NewPacket(n2.Address(), n1.Address(), "Hello World from p2!")
	// n2.SendPacket(p2)

	// for i := range 10 {
	// 	p := packet.NewPacket(n1.Address(), n2.Address(), fmt.Sprintf("Hello %v th packet", i))
	// 	n1.SendPacket(p)
	// }

	// n3を作成
	n3 := node.NewNode(3, "00:03", nes)
	l2 := node.NewLink(n1, n3, 1000, 0.01, 0.0, nes)
	l2.PrintLink()

	n4 := node.NewNode(4, "00:04", nes)
	l3 := node.NewLink(n2, n4, 1000, 0.01, 0.0, nes)
	l3.PrintLink()
	nes.Visualize()
	nes.Run()
}
