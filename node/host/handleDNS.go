package host

import (
	"fmt"
	"nt-simulator/address"
	"nt-simulator/packet"
)

func (n *host) processDNSPacket(dnsP *packet.DNSP) {
	if dnsP.GetMacHeader().DestinationMac.String() == n.MacAddress.String() {
		dp, err := dnsP.ParsePayload()
		if err != nil {
			fmt.Printf("dns parse error: %v\n", err)
			return
		}
		n.GetNES().LogPacketInfo(dnsP, "DNS Reply received", n.NodeId())
		if dp.QueryDomain != "" && dp.ResolvedIp != "" {
			n.onDNSReplyPacketReceived(dp.QueryDomain, dp.ResolvedIp)
		}
		return
	}
}

func (n *host) sendDNSQueryAndSetTraffic(destinationURL string, startTime float64, headerSize int, payloadSize int, protocol string) {
	n.waitingForDNSReply[destinationURL] = append(n.waitingForDNSReply[destinationURL], &dataWhenReceiveDNSReply{
		startTime:   startTime,
		headerSize:  headerSize,
		payloadSize: payloadSize,
		protocol:    protocol,
	})
	p := packet.NewDNSP(n.MacAddress, address.BroadcastMacAddress, n.IpAddress, address.NewIPAddress(n.dnsServerIp), n.GetNES().CurrentTime, destinationURL, packet.DNSQueryTypeA, "")
	n.GetNES().LogPacketInfo(p, "DNS Query", n.NodeId())
	n.internalSendPacket(p)
}

// DNSリプライを受信したら、待機中のパケットに対して処理を行う
func (n *host) onDNSReplyPacketReceived(query string, ipAddress string) {
	n.addDNSRecord(query, ipAddress)
	if _, ok := n.waitingForDNSReply[query]; ok {
		destinationIP := address.NewIPAddress(ipAddress)
		for _, v := range n.waitingForDNSReply[query] {
			n.setTraffic(destinationIP, v.startTime, v.headerSize, v.payloadSize, v.protocol)
		}
		n.waitingForDNSReply[query] = []*dataWhenReceiveDNSReply{}
	}
}

func (n *host) addDNSRecord(queryDomain string, resolvedIp string) {
	n.urlToIpMapping[queryDomain] = resolvedIp
}
