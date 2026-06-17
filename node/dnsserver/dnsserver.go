package dnsserver

import (
	"fmt"
	"nt-simulator/address"
	"nt-simulator/link"
	"nt-simulator/node/basenode"
	"nt-simulator/nteventsched"
	"nt-simulator/packet"
	"nt-simulator/packet/packetI"
)

type dnsserver struct {
	*basenode.BaseNode
	*address.MacAddress
	*address.IpAddress
	dnsRecords map[string]string
}

func NewDNSServer(nes *nteventsched.NtEventSched, nodeId int, ipAddress string) *dnsserver {
	ds := &dnsserver{
		BaseNode:   basenode.NewBaseNode(nodeId, nes),
		MacAddress: address.NewMacAddress(address.GenerateRandomMAC()),
		IpAddress:  address.NewIPAddress(ipAddress),
		dnsRecords: make(map[string]string),
	}
	nes.AddNode(ds)
	return ds
}

func (d *dnsserver) NodeColor() string { return "purple" }

func (d *dnsserver) ReceivePacket(p packetI.PacketI, l *link.Link) {
	if p.ArrivalTime() == -1 {
		d.GetNES().LogPacketInfo(p, "lost", d.NodeId())
		return
	}

	destMac := p.GetMacHeader().DestinationMac
	if destMac == nil {
		d.GetNES().LogPacketInfo(p, "dropped", d.NodeId())
		return
	}

	if destMac.String() == address.BroadcastMAC || destMac.String() == d.GetMacAddress().String() {
		if arpP, ok := p.(*packet.ArpP); ok {
			ap, err := arpP.ParsePayload()
			if err != nil {
				fmt.Printf("arp parse error: %v\n", err)
				return
			}
			if ap.Operation == packet.ArpOperationRequest && ap.DestIp == d.GetIPAddresses()[0].String() {
				d.sendArpReply(arpP)
				return
			}
		} else if dnsP, ok := p.(*packet.DNSP); ok {
			if dnsP.IpHeader.DestIp.String() == d.IpAddress.String() {
				d.GetNES().LogPacketInfo(p, "dns query received", d.NodeId())
				p.SetArrived(d.GetNES().CurrentTime)
				dnsResponsePacket := d.handleDNSQuery(dnsP)
				d.internalSendPacket(dnsResponsePacket)
			}
		} else {
			d.GetNES().LogPacketInfo(p, "dropped", d.NodeId())
		}
	}
}

func (d *dnsserver) GetIPAddresses() []*address.IpAddress {
	return []*address.IpAddress{d.IpAddress}
}

func (d *dnsserver) AddLink(link *link.Link, ip *address.IpAddress) {
	for _, l := range d.GetLinks() {
		if l == link {
			return
		}
	}
	d.SetLinks(append(d.GetLinks(), link))
}

func (d *dnsserver) AddDNSRecord(domain string, ip string) {
	d.dnsRecords[domain] = ip
}

func (d *dnsserver) handleDNSQuery(dnsP *packet.DNSP) *packet.DNSP {
	dp, err := dnsP.ParsePayload()
	if err != nil {
		panic(fmt.Sprintf("arp parse error: %v\n", err))
	}
	if resolvedIP, ok := d.dnsRecords[dp.QueryDomain]; ok {
		dnsResponsePacket := packet.NewDNSP(d.MacAddress, dnsP.GetMacHeader().SourceMac, d.IpAddress, dnsP.IpHeader.SourceIp, d.GetNES().CurrentTime, dp.QueryDomain, packet.DNSQueryTypeA, resolvedIP)
		return dnsResponsePacket
	}
	panic("ドメインに該当するIPアドレスが見つかりません")
}

func (d *dnsserver) sendArpReply(rp packetI.PacketI) {
	// 送られてきた元のノードに送り返す
	arpReplyPacket := packet.NewArpP(d.GetMacAddress(), rp.GetMacHeader().SourceMac, d.GetIPAddresses()[0], rp.GetIpHeader().SourceIp, d.GetNES().CurrentTime, packet.ArpOperationReply)
	d.GetNES().LogPacketInfo(arpReplyPacket, "ARP Reply", d.NodeId())
	d.internalSendPacket(arpReplyPacket)
}

func (d *dnsserver) internalSendPacket(p packetI.PacketI) {
	d.GetNES().LogPacketInfo(p, "sent", d.NodeId())
	for _, l := range d.GetLinks() {
		l.EnqueuePacket(p, d)
	}
}
