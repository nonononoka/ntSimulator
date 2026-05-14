package address

import (
	"net"
	"regexp"
)

type Address struct {
	macAddress string
	ipAddress  string
}

func NewAddress(macAddress string, ipAddress string) *Address {
	return &Address{macAddress: macAddress, ipAddress: ipAddress}
}

func (address *Address) IsValidMacAddress() bool {
	macAddress := address.macAddress
	matched, _ := regexp.MatchString(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`, macAddress)
	return matched
}

func (address *Address) IsValidV4Address() bool {
	ipAddress := address.ipAddress
	ip := net.ParseIP(ipAddress)
	return ip != nil && ip.To4() != nil
}

func (address *Address) IsValidCIDRNotation() bool {
	ipAddress := address.ipAddress
	_, _, err := net.ParseCIDR(ipAddress)
	if err != nil {
		return false
	}
	return true
}

func (address *Address) GetCIDRIpAddress() net.IP {
	ipAddress := address.ipAddress
	ip, _, err := net.ParseCIDR(ipAddress)
	if err != nil {
		panic("invalid CIDR notation: " + ipAddress)
	}
	return ip
}

func (address *Address) GetIPAddress() string {
	return address.ipAddress
}

func (address *Address) GetMacAddress() string {
	return address.macAddress
}
