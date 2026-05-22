package address

import (
	"crypto/rand"
	"fmt"
	"net"
	"net/netip"
)

type MacAddress struct {
	address string
}

type IpAddress struct {
	address string
}

type NetworkAddress struct {
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

func (address *IpAddress) GetNetworkAddress() net.IP {
	ipAddress := address.address
	ip, _, err := net.ParseCIDR(ipAddress)
	if err != nil {
		panic("invalid CIDR notation: " + ipAddress)
	}
	return ip
}

func (address *IpAddress) ConvertToNetworkCIDR() *IpAddress {
	prefix, err := netip.ParsePrefix(address.String())
	if err != nil {
		panic(fmt.Sprintf("無効なCIDR形式です: %w", err))
	}

	return NewIPAddress(prefix.Masked().String())
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

// GenerateRandomMAC はランダムなMACアドレスを生成して文字列で返します
func GenerateRandomMAC() string {
	// MACアドレスは6バイト（48ビット）
	buf := make([]byte, 6)

	// 暗号論的に安全なランダムなバイトを生成
	rand.Read(buf)

	// 1バイト目の下位2ビット目を1にすることで、
	// 「ローカルで管理されたアドレス（LAA）」であることを明示するのが一般的です。
	// ※完全なランダムでよければ、この1行は消しても動作します。
	buf[0] = (buf[0] | 2) & 0xfe

	// 16進数の文字列（コロン区切り）に整形
	macStr := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])

	return macStr
}
