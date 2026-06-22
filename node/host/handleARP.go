package host

import (
	"fmt"
	"nt-simulator/address"
	"nt-simulator/packet"
	"nt-simulator/packet/packetI"
)

type dataWhenReceiveArpReply struct {
	data            string
	sourcePort      int
	destinationPort int
	protocol        string
}

func (n *host) processARPPacket(arpP *packet.ArpP) {
	if arpP.GetMacHeader().DestinationMac.String() == address.BroadcastMacAddress.String() {
		ap, err := arpP.ParsePayload()
		if err != nil {
			fmt.Printf("arp parse error: %v\n", err)
			return
		}
		if ap.Operation == packet.ArpOperationRequest && ap.DestIp == n.GetIPAddresses()[0].String() {
			n.sendArpReply(arpP)
			return
		}
	}

	if arpP.GetMacHeader().DestinationMac.String() == n.MacAddress.String() && arpP.GetIpHeader().DestIp.String() == n.IpAddress.String() {
		ap, err := arpP.ParsePayload()
		if err != nil {
			fmt.Printf("arp parse error: %v\n", err)
			return
		}
		if ap.Operation == packet.ArpOperationReply && ap.DestIp == n.GetIPAddresses()[0].String() {
			n.GetNES().LogPacketInfo(arpP, "ARP Reply received", n.NodeId())
			sourceIp := address.NewIPAddress(ap.SourceIp)
			sourceMac := address.NewMacAddress(ap.SourceMac)
			n.AddToArpTable(sourceIp, sourceMac)
			n.onArpReplyPacketReceived(ap.SourceIp)
			return
		}
	}
}

func (n *host) sendArpRequest(ipAddress *address.IpAddress) {
	// ブロードキャスト
	arpPacket := packet.NewArpP(n.MacAddress, address.BroadcastMacAddress, n.IpAddress, ipAddress, n.GetNES().CurrentTime, packet.ArpOperationRequest)
	n.GetNES().LogPacketInfo(arpPacket, "ARP request", n.NodeId())
	n.internalSendPacket(arpPacket)
}

func (n *host) AddToArpTable(ipAddress *address.IpAddress, macAddress *address.MacAddress) {
	n.arpTable[ipAddress.String()] = macAddress
}

// arpリプライを受信したら、待機中のパケットに対して処理を行う
func (n *host) onArpReplyPacketReceived(ipAddress string) {
	if _, ok := n.waitingForArpReply[ipAddress]; ok {
		destinationIP := address.NewIPAddress(ipAddress)
		for _, v := range n.waitingForArpReply[ipAddress] {
			switch v.protocol {
			case "UDP":
				n.sendUDPPacket(destinationIP, v.data, v.sourcePort, v.destinationPort)
			case "TCP":
				n.startTCPConnectionAndSendPacket(destinationIP, v.data, v.sourcePort, v.destinationPort, n.GetNES().CurrentTime)
			}
		}
		n.waitingForArpReply[ipAddress] = []*dataWhenReceiveArpReply{}
	}
}

// arpリクエストを受け取って、宛先IPがこのノードのIPと一致していた場合の処理
func (n *host) sendArpReply(rp packetI.PacketI) {
	// 送られてきた元のノードに送り返す
	arpReplyPacket := packet.NewArpP(n.GetMacAddress(), rp.GetMacHeader().SourceMac, n.GetIPAddresses()[0], rp.GetIpHeader().SourceIp, n.GetNES().CurrentTime, packet.ArpOperationReply)
	n.GetNES().LogPacketInfo(arpReplyPacket, "ARP Reply", n.NodeId())
	n.internalSendPacket(arpReplyPacket)
}

func (n *host) getMacAddressFromIp(ipAddress *address.IpAddress) *address.MacAddress {
	macAddress, ok := n.arpTable[ipAddress.String()]
	if ok {
		return macAddress
	} else {
		return nil
	}
}

func (n *host) PrintArpTable() {
	fmt.Printf("--- ARP Table (%v) ---\n", n.NodeId()) // もしホスト名などがあれば
	fmt.Printf("%-15s   %-17s\n", "IP Address", "MAC Address")
	fmt.Println("---------------------------------------")

	if len(n.arpTable) == 0 {
		fmt.Println("(No entries found)")
		return
	}

	for ip, mac := range n.arpTable {
		// 左詰めで綺麗に並べて表示
		fmt.Printf("%-15s   %-17s\n", ip, mac)
	}
	fmt.Println("---------------------------------------")
}
