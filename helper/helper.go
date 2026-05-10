package helper

import (
	"net"
	"regexp"
)

func IsValidMacAddress(macAddress string) bool {
	matched, _ := regexp.MatchString(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`, macAddress)
	return matched
}

func IsValidV4Address(ipAddress string) bool {
	ip := net.ParseIP(ipAddress)
	return ip != nil && ip.To4() != nil
}
