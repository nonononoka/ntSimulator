package address

import (
	"fmt"
	"net"
)

type MacAddress struct {
	address string
}

type IpAddress struct {
	address string
}

func NewMacAddress(address string) *MacAddress {
	if !isValidMacAddress(address) {
		panic(fmt.Sprintf("not valid mac address: %s", address))
	}
	return &MacAddress{address: address}
}

// CIDR表記のアドレスを受け取るという想定
func NewIPAddress(address string) *IpAddress {
	if !isValidCIDRNotation(address) {
		panic(fmt.Sprintf("not valid ip address: %s", address))
	}
	return &IpAddress{address: address}
}

func isValidMacAddress(address string) bool {
	_, err := net.ParseMAC(address)
	return err == nil
}

func isValidCIDRNotation(address string) bool {
	_, _, err := net.ParseCIDR(address)
	if err != nil {
		return false
	}
	return true
}

func (address *IpAddress) GetCIDRIpAddress() net.IP {
	ipAddress := address.address
	ip, _, err := net.ParseCIDR(ipAddress)
	if err != nil {
		panic("invalid CIDR notation: " + ipAddress)
	}
	return ip
}

func (address *IpAddress) IsSameNetwork(otherIpAddress *IpAddress) bool {
	_, net1, _ := net.ParseCIDR(address.address)
	_, net2, _ := net.ParseCIDR(otherIpAddress.address)
	return net1.String() == net2.String()
}

func (address *IpAddress) String() string {
	return address.address
}

func (address *MacAddress) GetMacAddress() *MacAddress {
	return address
}

func (address *MacAddress) String() string {
	return address.address
}
