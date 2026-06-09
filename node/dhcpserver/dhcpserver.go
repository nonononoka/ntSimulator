package dhcpserver

import (
	"fmt"
	"net"
	"nt-simulator/address"
	"nt-simulator/link"
	"nt-simulator/node/basenode"
	"nt-simulator/nteventsched"
	"nt-simulator/packet"
	"nt-simulator/packet/packetI"
)

type dhcpserver struct {
	*basenode.BaseNode
	*address.MacAddress
	*address.IpAddress
	dnsServerIp   *address.IpAddress
	ipPoolUsedMap map[string]bool
}

func NewDHCPServer(nes *nteventsched.NtEventSched, nodeId int, ipAddress string, dnsServerIp string, startCIDR string) *dhcpserver {
	ds := &dhcpserver{
		BaseNode:      basenode.NewBaseNode(nodeId, nes),
		MacAddress:    address.NewMacAddress(address.GenerateRandomMAC()),
		IpAddress:     address.NewIPAddress(ipAddress),
		dnsServerIp:   address.NewIPAddress(dnsServerIp),
		ipPoolUsedMap: make(map[string]bool),
	}
	nes.AddNode(ds)
	ds.initializeIPPool(startCIDR)
	return ds
}

func (d *dhcpserver) NodeColor() string { return "brown" }

func (d *dhcpserver) ReceivePacket(p packetI.PacketI, l *link.Link) {
	if p.ArrivalTime() == -1 {
		d.GetNES().LogPacketInfo(p, "lost", d.NodeId())
		return
	}

	destMac := p.GetMacHeader().DestinationMac
	if destMac == nil {
		d.GetNES().LogPacketInfo(p, "dropped", d.NodeId())
		return
	}

	if destMac.String() == "FF:FF:FF:FF:FF:FF" || destMac.String() == d.GetMacAddress().String() {
		if arpP, ok := p.(*packet.ArpP); ok {
			ap, err := arpP.ParsePayload()
			if err != nil {
				fmt.Printf("arp parse error: %v\n", err)
				return
			}
			if ap.Operation == "request" && ap.DestIp == d.GetIPAddresses()[0].String() {
				d.sendArpReply(arpP)
				return
			}
		}
	}

	if destMac.String() == "FF:FF:FF:FF:FF:FF" && p.GetIpHeader().DestIp.String() == "255.255.255.255/32" {
		if dhcpP, ok := p.(*packet.DHCPP); ok {
			dhcppayload, err := dhcpP.ParsePayload()
			if err != nil {
				fmt.Printf("dhcp parse error: %v\n", err)
				return
			}

			switch dhcppayload.MessageType {
			case "DISCOVER":
				d.handleDHCPDiscover(dhcpP)
			case "REQUEST":
				d.handleDHCPRequest(dhcpP, &dhcppayload)
			}
		}
	}
}

func (d *dhcpserver) GetIPAddresses() []*address.IpAddress {
	return []*address.IpAddress{d.IpAddress}
}

func (d *dhcpserver) AddLink(link *link.Link, ip *address.IpAddress) {
	for _, l := range d.GetLinks() {
		if l == link {
			return
		}
	}
	d.SetLinks(append(d.GetLinks(), link))
}

// discoverパケットが来たら利用可能なIPアドレスを割り当ててDHCPOfferPacketを生成して送信
func (d *dhcpserver) handleDHCPDiscover(discoverPacket *packet.DHCPP) {
	assignedIP := d.getAvailableIp()
	offeredPacket := d.createDHCPOfferPacket(discoverPacket, assignedIP)
	d.internalSendPacket(offeredPacket)
}

func (d *dhcpserver) handleDHCPRequest(requestPacket *packet.DHCPP, payload *packet.DHCPPayload) {
	ackPacket := d.createDHCPACKPacket(requestPacket, payload)
	d.internalSendPacket(ackPacket)
}

func (d *dhcpserver) createDHCPOfferPacket(discoverPacket *packet.DHCPP, offeredIP string) *packet.DHCPP {
	dhcpOfferPacket := packet.NewDHCPPWithOfferedIP(d.GetMacAddress(), discoverPacket.MacHeader.SourceMac, d.GetIPAddresses()[0], discoverPacket.IpHeader.SourceIp, d.GetNES().CurrentTime, "OFFER", offeredIP)
	return dhcpOfferPacket
}

func (d *dhcpserver) createDHCPACKPacket(requestPacket *packet.DHCPP, payload *packet.DHCPPayload) *packet.DHCPP {
	assignedIP := payload.RequestedIP
	dhcpACKPacket := packet.NewDHCPPWithAssignedIPAndDNSIP(d.GetMacAddress(), requestPacket.MacHeader.SourceMac, d.GetIPAddresses()[0], requestPacket.IpHeader.SourceIp, d.GetNES().CurrentTime, "OFFER", assignedIP, d.dnsServerIp.String())
	return dhcpACKPacket
}

func (d *dhcpserver) getAvailableIp() string {
	for ip, used := range d.ipPoolUsedMap {
		if !used {
			d.ipPoolUsedMap[ip] = true
			return ip
		}
	}
	panic("使用可能なipが見つかりませんでした")
}

func (d *dhcpserver) initializeIPPool(startCIDR string) {
	if d.ipPoolUsedMap == nil {
		d.ipPoolUsedMap = make(map[string]bool)
	}

	ip, ipNet, err := net.ParseCIDR(startCIDR)
	if err != nil {
		fmt.Printf("CIDRのパースに失敗しました: %v\n", err)
		return
	}

	if ip.To4() == nil {
		fmt.Println("IPv4のCIDRを指定してください")
		return
	}

	currIP := make(net.IP, len(ipNet.IP))
	copy(currIP, ipNet.IP)

	// ループでIPをインクリメントしながらマップに追加
	for ipNet.Contains(currIP) {
		// ネットワークアドレスとブロードキャストアドレスを除外する場合の判定
		if !currIP.Equal(ipNet.IP) && !currIP.Equal(lastIP(ipNet)) {
			// 初期状態はすべて「未使用 (false)」
			d.ipPoolUsedMap[currIP.String()] = false
		}

		// 次のIPアドレスへ進める
		inc(currIP)
	}
}

// IPアドレスを1つ進める補助関数
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		if ip[j] > 0 {
			break
		}
	}
}

// ブロードキャストアドレスを計算する補助関数
func lastIP(ipNet *net.IPNet) net.IP {
	last := make(net.IP, len(ipNet.IP))
	for i := range ipNet.IP {
		last[i] = ipNet.IP[i] | ^ipNet.Mask[i]
	}
	return last
}

func (d *dhcpserver) sendArpReply(rp packetI.PacketI) {
	// 送られてきた元のノードに送り返す
	arpReplyPacket := packet.NewArpP(d.GetMacAddress(), rp.GetMacHeader().SourceMac, d.GetIPAddresses()[0], rp.GetIpHeader().SourceIp, d.GetNES().CurrentTime, "reply")
	d.GetNES().LogPacketInfo(arpReplyPacket, "ARP Reply", d.NodeId())
	d.internalSendPacket(arpReplyPacket)
}

func (d *dhcpserver) internalSendPacket(p packetI.PacketI) {
	d.GetNES().LogPacketInfo(p, "sent", d.NodeId())
	for _, l := range d.GetLinks() {
		l.EnqueuePacket(p, d)
	}
}
