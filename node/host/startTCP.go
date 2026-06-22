package host

import (
	"math/rand"
	"nt-simulator/address"
	"nt-simulator/packet"
)

func (n *host) startTCPConnectionAndSendPacket(destinationIp *address.IpAddress, data string, sourcePort int, destinationPort int, startTime float64) {
	tcpConnectionKey := n.createTCPConnectionKey(destinationIp.String(), destinationPort)
	segmentPayloadSize := n.mtu - 40 // IP header(20) + TCP header(20)
	// ここで、n.tcpConnectionsにkeyとvalueを登録
	if _, ok := n.tcpConnections[tcpConnectionKey]; !ok {
		n.initiateConnectionInfo(tcpConnectionKey, data, packet.TCPConnectionStateClosed, 1+rand.Intn(10000), 0)
		n.tcpConnections[tcpConnectionKey].trafficInfo = trafficInfo{
			endTime:     startTime + 1e6,
			payloadSize: segmentPayloadSize,
		}
	}
	// macアドレスがわからなかったら、arpリクエストを送信する
	destinationMac := n.getMacAddressFromIp(destinationIp) // destinationIPアドレスからmacアドレスをひく

	// 宛先IPアドレスに対応するMacアドレスが未知の場合、arpリクエストを送信して終わり
	if destinationMac == nil {
		// ARPリクエストを送信して、パケットを待機リストに追加する
		n.sendArpRequest(destinationIp)
		n.waitingForArpReply[destinationIp.String()] = append(n.waitingForArpReply[destinationIp.String()], &dataWhenReceiveArpReply{data: data, sourcePort: sourcePort, destinationPort: destinationPort, protocol: "TCP"})
		return
	}

	if !n.isTCPConnectionEstablished(destinationIp, destinationPort) {
		n.pendingTCPData[n.createTCPConnectionKey(destinationIp.String(), destinationPort)] = &pendingTCPData{data: data}
		n.initiateTCPHandshake(destinationIp, destinationMac, sourcePort, destinationPort)
	}
}

func (n *host) initiateTCPHandshake(destinationIp *address.IpAddress, destinationMac *address.MacAddress, sourcePort int, destinationPort int) {
	// establishedじゃなかったらSYNパケットを送信
	if !n.isTCPConnectionEstablished(destinationIp, destinationPort) {
		tcpConnectionKey := n.createTCPConnectionKey(destinationIp.String(), destinationPort)
		sequenceNumber := n.tcpConnections[tcpConnectionKey].sequenceNumber
		acknowledgementNumber := n.tcpConnections[tcpConnectionKey].acknowledgementNumber
		// flagsを"SYN"にして送信
		n.fragmentAndSendTCP(destinationMac, destinationIp, sourcePort, destinationPort, sequenceNumber, acknowledgementNumber, "SYN", "")
	}
}
