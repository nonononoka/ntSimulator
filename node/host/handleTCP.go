package host

import (
	"nt-simulator/address"
	"nt-simulator/packet"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// TCPパケットを受け取ったときの処理
func (n *host) processTCPPacket(p *packet.TCPP) {
	if p.GetMacHeader().DestinationMac.String() == n.MacAddress.String() {
		if p.GetIpHeader().DestIp.String() == n.IpAddress.String() {
			flags := strings.Split(p.TCPHeader.Flags, ",")
			if slices.Contains(flags, "SYN") && !slices.Contains(flags, "ACK") {
				// SYNパケットを受信した場合、SYN-ACKを送信
				n.sendTCPSYNACK(p)
				n.logTCPControlPacketProcessed(p)
			} else if slices.Contains(flags, "SYN") && slices.Contains(flags, "ACK") {
				// SYN-ACKパケットを受信した場合、ACKを送信して、接続完了したのでpendingDataも送信
				n.sendTCPACK(p)
				n.establishTCPConnection(p)
				n.logTCPControlPacketProcessed(p)
			} else if slices.Contains(flags, "ACK") {
				// ACKパケットを受信した場合、接続が確立されたとみなす
				n.establishTCPConnection(p)
				n.logTCPControlPacketProcessed(p)
			} else if slices.Contains(flags, "FIN") {
				// FINパケットを受信した場合、接続を修了
				n.terminateTCPConnection(p)
			} else {
				n.processDataPacket(p)
			}
		}
	}
}

func (n *host) logTCPControlPacketProcessed(p *packet.TCPP) {
	n.GetNES().LogPacketInfo(p, "arrived", n.NodeId())
	p.SetArrived(n.GetNES().CurrentTime)
	n.GetNES().LogPacketInfo(p, "processed", n.NodeId())
}

func (n *host) sendTCPSYNACK(p *packet.TCPP) {
	synAckPacketFlags := "SYN,ACK"
	n.sendTCPPacket(p.GetIpHeader().SourceIp, p.GetMacHeader().SourceMac, "", p.TCPHeader.DestinationPort, p.TCPHeader.SourcePort, synAckPacketFlags)
}

func (n *host) sendTCPACK(p *packet.TCPP) {
	ackPacketFlags := "ACK"
	n.sendTCPPacket(p.GetIpHeader().SourceIp, p.GetMacHeader().SourceMac, "", p.TCPHeader.DestinationPort, p.TCPHeader.SourcePort, ackPacketFlags)
}

func (n *host) startTCPConnectionAndSendPacket(destinationIp *address.IpAddress, data string, sourcePort int, destinationPort int, flags string) {
	destinationMac := n.getMacAddressFromIp(destinationIp) // destinationIPアドレスからmacアドレスをひく

	// 宛先IPアドレスに対応するMacアドレスが未知の場合、arpリクエストを送信して終わり
	if destinationMac == nil {
		// ARPリクエストを送信して、パケットを待機リストに追加する
		n.sendArpRequest(destinationIp)
		n.waitingForArpReply[destinationIp.String()] = append(n.waitingForArpReply[destinationIp.String()], &dataWhenReceiveArpReply{data: data, sourcePort: sourcePort, destinationPort: destinationPort, protocol: "TCP"})
		return
	}
	if !n.isTCPConnectionEstablished(destinationIp, destinationPort) {
		connectionKey := TCPConnectionKey{destinationIP: destinationIp.String(), destinationPort: destinationPort}
		n.pendingTCPData[connectionKey] = pendingTCPData{destinationIp: destinationIp.String(), destinationPort: destinationPort, sourcePort: sourcePort, data: data}
		// 接続が未確立なので、ハンドシェイクを開始。これは実際にはaccept関数がやってくれる
		n.initiateTCPHandshake(destinationIp, destinationMac, sourcePort, destinationPort)
	} else {
		n.sendTCPPacket(destinationIp, destinationMac, data, sourcePort, destinationPort, flags)
	}
}

func (n *host) sendTCPPacket(destinationIp *address.IpAddress, destinationMac *address.MacAddress, data string, sourcePort int, destinationPort int, flags string) {
	tcpHeaderSize := 20
	ipHeaderSize := 20
	headerSize := tcpHeaderSize + ipHeaderSize
	payloadSize := n.mtu - headerSize
	totalSize := len(data) // goだとこれはバイト数になる
	offset := 0
	originalDataId := uuid.New().String()

	// SYN/ACK などペイロードなしの制御パケットも1回は送信する
	if totalSize == 0 {
		p := packet.NewTCPPacket(n.MacAddress, destinationMac, n.IpAddress, destinationIp, 64, headerSize, n.GetNES().CurrentTime, originalDataId, false, 0, "", sourcePort, destinationPort, 0, 0, flags)
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
		p := packet.NewTCPPacket(n.MacAddress, destinationMac, n.IpAddress, destinationIp, 64, headerSize, n.GetNES().CurrentTime, originalDataId, moreFragment, fragmentOffset, fragmentData, sourcePort, destinationPort, 0, 0, flags)

		n.internalSendPacket(p) // 細かくfragmentにしたpacketを送信する
		offset += payloadSize
	}
}

// TCP接続を確立する。接続が確立されたら、保存しておいたデータを送信する
func (n *host) establishTCPConnection(p *packet.TCPP) {
	tcpConnectionKey := TCPConnectionKey{destinationIP: p.GetIpHeader().SourceIp.String(), destinationPort: p.TCPHeader.SourcePort}
	if n.isTCPConnectionEstablished(p.GetIpHeader().SourceIp, p.TCPHeader.SourcePort) {
		return
	}

	n.updateTCPConnectionState(p.GetIpHeader().SourceIp.String(), p.TCPHeader.SourcePort, packet.TCPConnectionStateEstablished)
	// 保存しておいたデータがあれば送信する
	if pendingData, ok := n.pendingTCPData[tcpConnectionKey]; ok {
		delete(n.pendingTCPData, tcpConnectionKey)
		n.sendTCPPacket(p.GetIpHeader().SourceIp, p.GetMacHeader().SourceMac, pendingData.data, pendingData.sourcePort, pendingData.destinationPort, "")
	}
}

func (n *host) isTCPConnectionEstablished(destinationIP *address.IpAddress, destinationPort int) bool {
	if status, ok := n.tcpConnections[TCPConnectionKey{destinationIP: destinationIP.String(), destinationPort: destinationPort}]; ok {
		if status == packet.TCPConnectionStateEstablished {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func (n *host) initiateTCPHandshake(destinationIp *address.IpAddress, destinationMac *address.MacAddress, sourcePort int, destinationPort int) {
	// establishedじゃなかったらSYNパケットを送信
	if !n.isTCPConnectionEstablished(destinationIp, destinationPort) {
		// 接続状態をSYN SENTに更新
		n.updateTCPConnectionState(destinationIp.String(), destinationPort, packet.TCPConnectionStateSynSent)
		// flagsを"SYN"にして送信
		n.sendTCPPacket(destinationIp, destinationMac, "", sourcePort, destinationPort, "SYN")
	}
}

// 指定された宛先に対するTCP接続の状態を更新
func (n *host) updateTCPConnectionState(destinationIP string, destinatioPort int, newState packet.TCPConnectionState) {
	n.tcpConnections[TCPConnectionKey{destinationIP: destinationIP, destinationPort: destinatioPort}] = newState
}

func (n *host) terminateTCPConnection(p *packet.TCPP) {
	tcpConnectionKey := TCPConnectionKey{destinationIP: p.GetIpHeader().SourceIp.String(), destinationPort: p.TCPHeader.SourcePort}
	delete(n.tcpConnections, tcpConnectionKey)
}
