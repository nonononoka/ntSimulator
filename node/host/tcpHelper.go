package host

import (
	"nt-simulator/address"
	"nt-simulator/packet"

	"github.com/google/uuid"
)

// startTCP.goとhandleTCP.goの二つにあるやつ

// TCPハンドシェイク済みのTCPパケットを実際に送信するフェーズ
func (n *host) sendTCPPacket(destinationIp *address.IpAddress, destinationMac *address.MacAddress, data string, sourcePort int, destinationPort int, flags string, sequenceNumber int, acknowledgementNumber int) {
	// 宛先IPアドレスに対応するMacアドレスが未知の場合、arpリクエストを送信して終わり
	if destinationMac == nil {
		// ARPリクエストを送信して、パケットを待機リストに追加する
		n.sendArpRequest(destinationIp)
		n.waitingForArpReply[destinationIp.String()] = append(n.waitingForArpReply[destinationIp.String()], &dataWhenReceiveArpReply{data: data, sourcePort: sourcePort, destinationPort: destinationPort, protocol: "TCP"})
		return
	}

	// _send_tcp_packetに相当
	// SYN/ACK などペイロードなしの制御パケットも1回は送信する
	n.fragmentAndSendTCP(destinationMac, destinationIp, sourcePort, destinationPort, sequenceNumber, acknowledgementNumber, flags, data)
}

func (n *host) fragmentAndSendTCP(destinationMac *address.MacAddress, destinationIp *address.IpAddress, sourcePort int, destinationPort int, sequenceNumber int, acknowledgementNumber int, flags string, data string) {
	tcpHeaderSize := 20
	ipHeaderSize := 20
	headerSize := tcpHeaderSize + ipHeaderSize
	payloadSize := n.mtu - headerSize
	totalSize := len(data) // goだとこれはバイト数になる
	offset := 0
	originalDataId := uuid.New().String()

	// _send_tcp_packetに相当
	// SYN/ACK などペイロードなしの制御パケットも1回は送信する
	if totalSize == 0 {
		p := packet.NewTCPPacket(n.MacAddress, destinationMac, n.IpAddress, destinationIp, 64, headerSize, n.GetNES().CurrentTime, originalDataId, false, 0, "", sourcePort, destinationPort, sequenceNumber, acknowledgementNumber, flags)
		n.internalSendPacket(p)
		return
	}

	for offset < totalSize {
		moreFragment := (offset + payloadSize) < totalSize

		end := offset + payloadSize
		if end > totalSize {
			end = totalSize
		}
		fragmentData := data[offset:end]
		fragmentOffset := offset
		p := packet.NewTCPPacket(n.MacAddress, destinationMac, n.IpAddress, destinationIp, 64, headerSize, n.GetNES().CurrentTime, originalDataId, moreFragment, fragmentOffset, fragmentData, sourcePort, destinationPort, sequenceNumber, acknowledgementNumber, flags)

		n.internalSendPacket(p) // 細かくfragmentにしたpacketを送信する
		offset += payloadSize
	}
}

// デフォルトは、state = "CLOSED, sequenceNumberは0で、acknowkedgementNumberは0、dataは"""
func (n *host) initiateConnectionInfo(connectionKey tcpConnectionKey, data string, state packet.TCPConnectionState, sequenceNumber int, acknowledgementNUmber int) {
	n.tcpConnections[connectionKey] = &tcpConnectionStateValue{
		data:                    data,
		TCPConnectionState:      state,
		sequenceNumber:          sequenceNumber,
		acknowledgementNumber:   acknowledgementNUmber,
		receivedSequenceNumbers: make(map[int]struct{}),
	}
}

func (n *host) logTCPControlPacketProcessed(p *packet.TCPP) {
	n.GetNES().LogPacketInfo(p, "arrived", n.NodeId())
	p.SetArrived(n.GetNES().CurrentTime)
	n.GetNES().LogPacketInfo(p, "processed", n.NodeId())
}

func (n *host) isTCPConnectionEstablished(destinationIP *address.IpAddress, destinationPort int) bool {
	if status, ok := n.tcpConnections[n.createTCPConnectionKey(destinationIP.String(), destinationPort)]; ok {
		if status.TCPConnectionState == packet.TCPConnectionStateEstablished {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

// 指定された宛先に対するTCP接続の状態を更新
func (n *host) updateTCPConnectionState(tcpConnectionKey tcpConnectionKey, newState packet.TCPConnectionState) {
	if _, ok := n.tcpConnections[tcpConnectionKey]; !ok {
		n.initiateConnectionInfo(tcpConnectionKey, "", newState, 0, 0)
	} else {
		n.tcpConnections[tcpConnectionKey].TCPConnectionState = newState
	}
}

func (n *host) createTCPConnectionKeyFromPacket(p *packet.TCPP) tcpConnectionKey {
	connectionKey := tcpConnectionKey{destinationIP: p.GetIpHeader().SourceIp.String(), destinationPort: p.TCPHeader.SourcePort}
	return connectionKey
}

func (n *host) createTCPConnectionKey(destinationIP string, destinatioPort int) tcpConnectionKey {
	connectionKey := tcpConnectionKey{destinationIP: destinationIP, destinationPort: destinatioPort}
	return connectionKey
}
