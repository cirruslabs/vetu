package subnetfinder

import (
	"fmt"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	"net"
)

func FindAvailableSubnet(prefixLen ipaddr.BitCount) (net.IP, net.IP, net.IP, net.IPNet, error) {
	// Create a trie that will contain the address space available to us
	availableAddressSpace := ipaddr.NewTrie[*ipaddr.IPv4Address]()

	// Add private address space (as defined in RFC 1918[1]) to the trie
	//
	// [1]: https://datatracker.ietf.org/doc/html/rfc1918#section-3
	privateAddressSpaceRaw := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, item := range privateAddressSpaceRaw {
		availableAddressSpace.Add(ipaddr.NewIPAddressString(item).GetAddress().ToIPv4())
	}

	// Subtract address space that is already utilized on the host from the trie
	interfaceAddrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, nil, nil, net.IPNet{}, err
	}

	for _, interfaceAddrUncooked := range interfaceAddrs {
		interfaceAddr := ipaddr.NewIPAddressString(interfaceAddrUncooked.String()).GetAddress().ToIPv4()

		// IPv4 only for now
		if interfaceAddr == nil {
			continue
		}

		// If the interface uses a network that consumes node(s) of our trie,
		// remove these node(s) to signify that they're unavailable for use
		availableAddressSpace.RemoveElementsContainedBy(interfaceAddr)

		// If the interface uses a network that is covered by our trie, subtract
		// that network, removing the covering node and re-inserting the result
		// of subtraction of the interface network from the covering node
		if match := availableAddressSpace.LongestPrefixMatch(interfaceAddr); match != nil {
			splits := match.Subtract(interfaceAddr)

			availableAddressSpace.Remove(match)

			for _, prefixBlock := range splits[0].MergeToPrefixBlocks(splits[1:]...) {
				availableAddressSpace.Add(prefixBlock)
			}
		}
	}

	// Iterate through our trie until we're able to
	// get a subnet of the desired length
	trieIter := availableAddressSpace.Iterator()

	for trieIter.HasNext() {
		availableSubnet := trieIter.Next()

		// We won't be able to carve out a /29 out of /30
		if prefixLen < availableSubnet.GetMinPrefixLenForBlock() {
			continue
		}

		availableSubnetIter := availableSubnet.SetPrefixLen(prefixLen).PrefixBlockIterator()

		if !availableSubnetIter.HasNext() {
			continue
		}

		desiredSubnet := availableSubnetIter.Next()

		desiredSubnetIter := desiredSubnet.Iterator()

		_ = desiredSubnetIter.Next() // skip network
		firstHost := desiredSubnetIter.Next()
		secondHost := desiredSubnetIter.Next()
		thirdHost := desiredSubnetIter.Next()

		return firstHost.GetNetIP(), secondHost.GetNetIP(), thirdHost.GetNetIP(), net.IPNet{
			IP:   desiredSubnet.GetNetIP(),
			Mask: desiredSubnet.GetNetworkMask().Bytes(),
		}, nil
	}

	return nil, nil, nil, net.IPNet{},
		fmt.Errorf("no available subnet with prefix length of %d is found", prefixLen)
}
