package host

import (
	"fmt"
	"math/rand/v2"
	"nt-simulator/address"
	"nt-simulator/packet"
)

func (n *host) processDHCPPacket(dhcpP *packet.DHCPP) {
	if dhcpP.GetMacHeader().DestinationMac.String() == n.MacAddress.String() {
		dhcpPayload, err := dhcpP.ParsePayload()
		if err != nil {
			fmt.Printf("dhcp parse error: %v\n", err)
			return
		}
		switch dhcpPayload.MessageType {
		case packet.DHCPMessageTypeOffer:
			n.GetNES().LogPacketInfo(dhcpP, fmt.Sprintf("DHCP offer received: %s", dhcpPayload.OfferedIP), n.NodeId())
			n.sendDHCPRequest(dhcpPayload.OfferedIP)
			return
		case packet.DHCPMessageTypeACK:
			n.GetNES().LogPacketInfo(dhcpP, fmt.Sprintf("DHCP ack received: %s", dhcpPayload.AssignedIP), n.NodeId())
			n.IpAddress = address.NewIPAddress(dhcpPayload.AssignedIP)
			n.dnsServerIp = dhcpPayload.DnsServerIP
			return
		}
	}
}

func (n *host) scheduleDHCPPacket() {
	if n.IpAddress.IsNetworkAddress() {
		randomDelay := 0.5 + rand.Float64()*0.1
		n.GetNES().ScheduleEvent(n.GetNES().CurrentTime+randomDelay, func(args ...any) { n.sendDHCPDiscover() })
	}
}

func (n *host) sendDHCPDiscover() {
	dhcpDiscoverPacket := packet.NewDHCPP(n.MacAddress, address.BroadcastMacAddress, address.DHCPSourceIPAddress, address.BroadcastIPAddress, n.GetNES().CurrentTime, packet.DHCPMessageTypeDiscover)
	n.internalSendPacket(dhcpDiscoverPacket)
}

func (n *host) sendDHCPRequest(requestedIP string) {
	dhcpRequestPacket := packet.NewDHCPPWithRequestedIP(n.MacAddress, address.BroadcastMacAddress, address.DHCPSourceIPAddress, address.BroadcastIPAddress, n.GetNES().CurrentTime, packet.DHCPMessageTypeRequest, requestedIP)
	n.internalSendPacket(dhcpRequestPacket)
}
