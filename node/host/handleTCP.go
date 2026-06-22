package host

import (
	"math/rand"
	"nt-simulator/packet"
	"slices"
	"strings"
)

type trafficInfo struct {
	endTime     float64
	payloadSize int
}

// 送信側→data, trafficInfo, state, sequenceNumber
// 受信側→receivedSequenceNumbers, trafficInfo, state, acknowledgementNumber
type tcpConnectionStateValue struct {
	data string
	trafficInfo
	packet.TCPConnectionState
	sequenceNumber          int
	receivedSequenceNumbers map[int]struct{} // 受け取ったsequenceNumberたち。setがないのでこう書いている
	acknowledgementNumber   int              // acknowledgementNumber未満のやつを受け取った
}

type tcpConnectionKey struct {
	destinationIP   string
	destinationPort int
}

type pendingTCPData struct {
	destinationIp   string
	data            string
	sourcePort      int
	destinationPort int
}

// TCPパケットを受け取ったときの処理
func (n *host) processTCPPacket(p *packet.TCPP) {
	if p.GetMacHeader().DestinationMac.String() == n.MacAddress.String() {
		if p.GetIpHeader().DestIp.String() == n.IpAddress.String() {
			flags := strings.Split(p.TCPHeader.Flags, ",")
			if slices.Contains(flags, "SYN") {
				if slices.Contains(flags, "ACK") {
					// SYN-ACKを受信した場合、ACKを返信してパケットを送信
					// stateを更新して、ack番号をseq番号+1にする
					n.establishTCPConnection(p)
					n.sendTCPACK(p)
					n.sendTCPDataPacket(p)
					n.logTCPControlPacketProcessed(p)
				} else {
					// SYNパケットを受信した場合、SYN-ACKを送信
					n.sendTCPSYNACK(p)
					n.logTCPControlPacketProcessed(p)
				}
				return
			}

			if slices.Contains(flags, "ACK") {
				n.handleAcknowledgement(p)
			}

			// 普通のデータTCPパケットが来たとき
			if slices.Contains(flags, "PSH") {
				// ACK番号を更新
				n.updateACKNumber(p)
				// ACKを返信
				n.sendTCPACK(p)
				n.processDataPacket(p)
			}

			if slices.Contains(flags, "FIN") {
				// FINパケットを受信した場合、接続を修了
				n.terminateTCPConnection(p)
			}
		}
	}
}

// 実際のdataが入ったTCPパケットを送信
func (n *host) sendTCPDataPacket(p *packet.TCPP) {
	println("send tcp data packet")
	connectionKey := n.createTCPConnectionKeyFromPacket(p)
	if _, ok := n.tcpConnections[connectionKey]; !ok {
		return
	}
	trafficInfo := n.tcpConnections[connectionKey].trafficInfo
	if n.GetNES().CurrentTime < trafficInfo.endTime {
		for {
			if n.windows[connectionKey] >= n.windowSize {
				break
			}
			remainingData := n.tcpConnections[connectionKey].data
			if len(remainingData) == 0 {
				return
			}
			payloadSize := trafficInfo.payloadSize
			if payloadSize > len(remainingData) {
				payloadSize = len(remainingData)
			}
			dataToSend := remainingData[:payloadSize]

			n.sendTCPPacket(p.GetIpHeader().SourceIp, p.GetMacHeader().SourceMac, dataToSend, p.TCPHeader.DestinationPort, p.TCPHeader.SourcePort, "PSH", n.tcpConnections[connectionKey].sequenceNumber, n.tcpConnections[connectionKey].acknowledgementNumber)
			n.tcpConnections[connectionKey].data = remainingData[payloadSize:]
			// sequenceNumberは、これ以降を送るよってことなので、次送るときのために更新する
			n.tcpConnections[connectionKey].sequenceNumber += len(dataToSend)

			if len(n.tcpConnections[connectionKey].data) == 0 {
				break
			}
		}
	}
}

func (n *host) handleAcknowledgement(p *packet.TCPP) {
	connectionKey := n.createTCPConnectionKeyFromPacket(p)
	// ackNumber := p.TCPHeader.AcknowledgementNumber

	if _, ok := n.tcpConnections[connectionKey]; !ok {
		return
	}

	if _, ok := n.windows[connectionKey]; !ok {
		// TODOここはよくわからん
		n.windows[connectionKey] = 0
	}
}

// 来たシーケンス番号を見て、シーケンス番号からpayloadLength分は受け取ったので
// ack番号を更新する
func (n *host) updateACKNumber(p *packet.TCPP) {
	connectionKey := n.createTCPConnectionKeyFromPacket(p)
	// sequenceNumberからpayloadLengthまで送ったよ
	receivedSequenceNumber := p.TCPHeader.SequenceNumber
	payloadLength := len(p.Payload)

	// ack番号未満まで届いてる
	currentACKNumber := n.tcpConnections[connectionKey].acknowledgementNumber
	// 今まで届いた番号たち
	receivedSequenceNumbers := n.tcpConnections[connectionKey].receivedSequenceNumbers

	// 受信したシーケンス番号をセットに追加
	for i := receivedSequenceNumber; i < receivedSequenceNumber+payloadLength; i++ {
		receivedSequenceNumbers[i] = struct{}{}
	}

	// 期待する次のシーケンス番号を見つける
	nextExpectedSeq := currentACKNumber
	for {
		if _, ok := receivedSequenceNumbers[nextExpectedSeq]; ok {
			nextExpectedSeq++
		} else {
			break
		}
	}

	// ACK番号を更新
	if nextExpectedSeq != currentACKNumber {
		// ack番号未満まで届いてるよ
		n.tcpConnections[connectionKey].acknowledgementNumber = nextExpectedSeq
	}
}

// SYNが来たときに返すやつ
func (n *host) sendTCPSYNACK(p *packet.TCPP) {
	tcpConnectionKey := n.createTCPConnectionKeyFromPacket(p)
	sequenceNumber := 1 + rand.Intn(10000)
	acknowledgementNumber := p.TCPHeader.SequenceNumber + 1

	// 受信側は、コネクション情報を登録
	if _, ok := n.tcpConnections[tcpConnectionKey]; !ok {
		n.initiateConnectionInfo(tcpConnectionKey, "", packet.TCPConnectionStateSynReceived, sequenceNumber, acknowledgementNumber)
	}
	synAckPacketFlags := "SYN,ACK"
	n.sendTCPPacket(p.GetIpHeader().SourceIp, p.GetMacHeader().SourceMac, "", p.TCPHeader.DestinationPort, p.TCPHeader.SourcePort, synAckPacketFlags, sequenceNumber, acknowledgementNumber)
}

// SYN-ACKが来たときに返すやつ
func (n *host) sendTCPACK(p *packet.TCPP) {
	tcpConnectionKey := n.createTCPConnectionKeyFromPacket(p)
	if _, ok := n.tcpConnections[tcpConnectionKey]; ok {
		ackPacketFlags := "ACK"
		n.sendTCPPacket(p.GetIpHeader().SourceIp, p.GetMacHeader().SourceMac, "", p.TCPHeader.DestinationPort, p.TCPHeader.SourcePort, ackPacketFlags, n.tcpConnections[tcpConnectionKey].sequenceNumber, n.tcpConnections[tcpConnectionKey].acknowledgementNumber)
	} else {
		panic("Error: Connection key not found in tcp_connections.")
	}
}

// TCP接続を確立する。接続が確立されたら、保存しておいたデータを送信する
func (n *host) establishTCPConnection(p *packet.TCPP) {
	tcpConnectionKey := n.createTCPConnectionKeyFromPacket(p)
	n.tcpConnections[tcpConnectionKey].TCPConnectionState = packet.TCPConnectionStateEstablished
	n.tcpConnections[tcpConnectionKey].acknowledgementNumber = p.TCPHeader.SequenceNumber + 1
	// 送信済みSYN分のシーケンス番号を進める
	n.tcpConnections[tcpConnectionKey].sequenceNumber++
}

func (n *host) terminateTCPConnection(p *packet.TCPP) {
	delete(n.tcpConnections, n.createTCPConnectionKeyFromPacket(p))
}
