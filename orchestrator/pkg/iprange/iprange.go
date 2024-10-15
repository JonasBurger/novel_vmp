package iprange

import (
	"bytes"
	"fmt"
	"net"
	"strings"
)

// IPRange represents a range of IP addresses
type IPRange struct {
	Start net.IP
	End   net.IP
}

// NewIPRangeFromString creates an IPRange from a string
// The string can be in the format "startIP - endIP", CIDR notation like "127.0.0.0/24", or a single IP address
func NewIPRangeFromString(ipRangeStr string) (*IPRange, error) {
	ipRangeStr = strings.TrimSpace(ipRangeStr)

	if strings.Contains(ipRangeStr, "/") {
		return newIPRangeFromCIDR(ipRangeStr)
	}

	if strings.Contains(ipRangeStr, "-") {
		return newIPRangeFromIPRange(ipRangeStr)
	}

	return newIPRangeFromSingleIP(ipRangeStr)
}

// newIPRangeFromIPRange creates an IPRange from a "startIP - endIP" string
func newIPRangeFromIPRange(ipRangeStr string) (*IPRange, error) {
	parts := strings.Split(ipRangeStr, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid IP range format")
	}

	startIP := net.ParseIP(strings.TrimSpace(parts[0]))
	endIP := net.ParseIP(strings.TrimSpace(parts[1]))

	if startIP == nil || endIP == nil {
		return nil, fmt.Errorf("invalid IP address in range")
	}

	return &IPRange{Start: startIP, End: endIP}, nil
}

// newIPRangeFromCIDR creates an IPRange from a CIDR notation string
func newIPRangeFromCIDR(cidrStr string) (*IPRange, error) {
	ip, ipnet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR notation: %v", err)
	}

	startIP := ip.Mask(ipnet.Mask)
	endIP := make(net.IP, len(startIP))
	copy(endIP, startIP)
	for i := len(endIP) - 1; i >= 0; i-- {
		endIP[i] |= ^ipnet.Mask[i]
	}

	return &IPRange{Start: startIP, End: endIP}, nil
}

// newIPRangeFromSingleIP creates an IPRange from a single IP address
func newIPRangeFromSingleIP(ipStr string) (*IPRange, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address")
	}
	return &IPRange{Start: ip, End: ip}, nil
}

// Contains checks if the given IP is within the IP range
func (r *IPRange) Contains(ip net.IP) bool {
	if ip.To4() == nil {
		return false
	}
	return bytesCompare(ip, r.Start) >= 0 && bytesCompare(ip, r.End) <= 0
}

// bytesCompare compares two IP addresses as byte slices
func bytesCompare(a, b net.IP) int {
	return bytes.Compare(a.To4(), b.To4())
}

// String converts the IPRange to a string representation
func (r *IPRange) String() string {
	return fmt.Sprintf("%s - %s", r.Start.String(), r.End.String())
}

// NextIP returns the next IP address in the range. It returns nil if the end of the range is reached.
func (r *IPRange) NextIP(currentIP net.IP) net.IP {
	nextIP := make(net.IP, len(currentIP))
	copy(nextIP, currentIP)
	for i := len(nextIP) - 1; i >= 0; i-- {
		nextIP[i]++
		if nextIP[i] > 0 {
			break
		}
	}

	if !r.Contains(nextIP) {
		return nil
	}
	return nextIP
}

// Iterate iterates over all IPs in the IP range and sends them to the provided channel
func (r *IPRange) Iterate() <-chan net.IP {
	ch := make(chan net.IP)
	go func() {
		defer close(ch)
		for ip := r.Start; ip != nil && r.Contains(ip); ip = r.NextIP(ip) {
			ch <- ip
		}
	}()
	return ch
}

// AllIPs returns a slice of all IPs in the IP range
func (r *IPRange) AllIPs() []net.IP {
	var ips []net.IP
	for ip := range r.Iterate() {
		ips = append(ips, ip)
	}
	return ips
}
