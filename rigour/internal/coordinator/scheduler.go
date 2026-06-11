package coordinator

import (
	"fmt"
	"net"
)

func SplitInto24(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}

	ones, _ := ipnet.Mask.Size()
	if ones > 24 {
		return []string{cidr}, nil
	}

	var subnets []string
	numSubnets := 1 << (24 - ones)

	baseIP := ip.Mask(ipnet.Mask)
	for i := 0; i < numSubnets; i++ {
		newIP := make(net.IP, len(baseIP))
		copy(newIP, baseIP)

		// Increment third octet (for IPv4 /24 split)
		if len(newIP) == 16 { // IPv6 mapped or actual IPv6
			ipv4 := newIP.To4()
			if ipv4 != nil {
				ipv4[2] += byte(i)
				newIP = ipv4
			}
		} else {
			newIP[2] += byte(i)
		}

		subnets = append(subnets, fmt.Sprintf("%s/24", newIP.String()))
	}

	return subnets, nil
}
