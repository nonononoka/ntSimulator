package packet

import (
	"nt-simulator/address"
	"nt-simulator/packet/packetI"

	"github.com/google/uuid"
)

type TCPConnectionState string

const (
	TCPConnectionStateSynReceived TCPConnectionState = "SYN_RECEIVED"
	TCPConnectionStateClosed      TCPConnectionState = "CLOSED"
	TCPConnectionStateSynSent     TCPConnectionState = "SYN_SENT"
	TCPConnectionStateEstablished TCPConnectionState = "ESTABLISHED"
)

type TCPHeader struct {
	SourcePort            int
	DestinationPort       int
	SequenceNumber        int
	AcknowledgementNumber int
	Flags                 string
}

type TCPP struct {
	Packet
	TCPHeader
}

func NewTCPPacket(s *address.MacAddress, d *address.MacAddress, sourceip *address.IpAddress, destip *address.IpAddress, ttl int, header_size int, currentTime float64, originalDataId string, morefragment bool, offset int, p string, sourcePort int, destinationPort int, sequenceNumber int, acknowledgementNumber int, flags string) *TCPP {
	size := header_size + len(p)
	return &TCPP{
		Packet: Packet{
			MacHeader: packetI.MacHeader{
				SourceMac:      s,
				DestinationMac: d,
			},
			IpHeader: packetI.IpHeader{
				SourceIp:       sourceip,
				DestIp:         destip,
				TTL:            ttl,
				FragmentFlags:  packetI.FragmentFlags{OriginalDataId: originalDataId, MoreFragment: morefragment},
				FragmentOffset: offset,
			},
			Payload:      p,
			Size:         size,
			Id:           uuid.New().String(),
			creationTime: currentTime,
		},
		TCPHeader: TCPHeader{
			SourcePort:            sourcePort,
			DestinationPort:       destinationPort,
			SequenceNumber:        sequenceNumber,
			AcknowledgementNumber: acknowledgementNumber,
			Flags:                 flags,
		},
	}
}
