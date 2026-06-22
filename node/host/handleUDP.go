package host

import (
	"nt-simulator/address"
	"nt-simulator/packet"

	"github.com/google/uuid"
)

func (n *host) processUDPPacket(p *packet.UDPP) {
	if p.GetMacHeader().DestinationMac.String() == n.MacAddress.String() && p.GetIpHeader().DestIp.String() == n.IpAddress.String() {
		n.processDataPacket(p)
	}
}

func (n *host) sendUDPPacket(destinationIp *address.IpAddress, data string, sourcePort int, destinationPort int) {
	destinationMac := n.getMacAddressFromIp(destinationIp) // destinationIPアドレスからmacアドレスをひく

	// 宛先IPアドレスに対応するMacアドレスが未知の場合、arpリクエストを送信して終わり
	if destinationMac == nil {
		// ARPリクエストを送信して、パケットを待機リストに追加する
		n.sendArpRequest(destinationIp)
		n.waitingForArpReply[destinationIp.String()] = append(n.waitingForArpReply[destinationIp.String()], &dataWhenReceiveArpReply{data: data, sourcePort: sourcePort, destinationPort: destinationPort, protocol: "UDP"})
		return
	}
	udpHeaderSize := 8
	ipHeaderSize := 20
	headerSize := udpHeaderSize + ipHeaderSize
	payloadSize := n.mtu - headerSize
	totalSize := len(data) // goだとこれはバイト数になる
	offset := 0
	originalDataId := uuid.New().String()

	for offset < totalSize {
		moreFragment := (offset + payloadSize) < totalSize

		end := offset + payloadSize
		if end > totalSize {
			end = totalSize
		}
		fragmentData := data[offset:end]
		fragmentOffset := offset
		p := packet.NewUDPPacket(n.MacAddress, destinationMac, n.IpAddress, destinationIp, 64, headerSize, n.GetNES().CurrentTime, originalDataId, moreFragment, fragmentOffset, fragmentData, sourcePort, destinationPort)

		n.internalSendPacket(p) // 細かくfragmentにしたpacketを送信する
		offset += payloadSize
	}
}
