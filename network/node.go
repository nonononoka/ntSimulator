package network

import (
	"net"
	"nt-simulator/packet"
	"regexp"
)

// networkに含まれるnode（terminal nodeとかswitchとか含めて）のinterface
type node interface {
	PrintNode()
	NodeId() int
	AddLink(link *Link)
	receivePacket(p packet.PacketI, l *Link)
}

func isValidMacAddress(macAddress string) bool {
	matched, _ := regexp.MatchString(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`, macAddress)
	return matched
}

func isValidIpV4Address(ipAddress string) bool {
	ip := net.ParseIP(ipAddress)
	return ip != nil && ip.To4() != nil
}
