package packet

import (
	"nt-simulator/address"
	"nt-simulator/packet/packetI"

	"github.com/google/uuid"
)

type UDPHeader struct {
	SourcePort      int
	DestinationPort int
}

type UDPP struct {
	Packet
	UDPHeader
}

func NewUDPPacket(s *address.MacAddress, d *address.MacAddress, sourceip *address.IpAddress, destip *address.IpAddress, ttl int, header_size int, currentTime float64, originalDataId string, morefragment bool, offset int, p string, sourcePort int, destinationPort int) *UDPP {
	size := header_size + len(p)

	return &UDPP{
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
		UDPHeader: UDPHeader{
			SourcePort:      sourcePort,
			DestinationPort: destinationPort,
		},
	}
}
