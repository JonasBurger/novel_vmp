package scheduler

import (
	"bufio"
	"log"
	"net"
	"os"

	"github.com/spf13/viper"
	"my.org/novel_vmp/data"
	"my.org/novel_vmp/pkg/iprange"
)

type Scope struct {
	ips     []iprange.IPRange
	domains []string

	excluedIps      []iprange.IPRange
	excludedDomains []string
}

var CurrentScope *Scope

func NewScopeFromViperConfig() *Scope {
	println("looking up the ips of the domains declared in scope. Might take a while depending on the amount of domains.")

	scope := &Scope{}
	scope.ips = parseIPRanges(viper.GetStringSlice("scope.ips"))
	scope.domains = viper.GetStringSlice("scope.domains")

	scope.excluedIps = parseIPRanges(viper.GetStringSlice("scope.excluded_ips"))
	scope.excludedDomains = viper.GetStringSlice("scope.excluded_domains")

	if CurrentScope != nil {
		log.Panicf("There already exists a Scope instance.")
	}
	CurrentScope = scope

	file, err := os.Open("domainlist.txt")
	if err != nil {
		log.Fatalf("failed to open file: %s", err)
	}
	defer file.Close()
	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Create a slice to store the domains
	var domains []string

	// Scan the file line by line
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" && scope.IsDomainsIPInScope(line) { // Check if the line is not empty
			domains = append(domains, scanner.Text())
		}
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		log.Fatalf("failed to scan file: %s", err)
	}
	scope.domains = append(scope.domains, domains...)

	return scope
}

func parseIPRanges(iprangesStrings []string) []iprange.IPRange {
	var ranges []iprange.IPRange
	for _, iprangeString := range iprangesStrings {
		ipRange, err := iprange.NewIPRangeFromString(iprangeString)
		if err != nil {
			log.Fatalf("Error parsing IP range: %v", err)
		}
		ranges = append(ranges, *ipRange)
	}
	return ranges
}

func (s *Scope) IsIPInScope(ip net.IP) bool {
	for _, excludedRange := range s.excluedIps {
		if excludedRange.Contains(ip) {
			return false
		}
	}
	for _, includedRange := range s.ips {
		if includedRange.Contains(ip) {
			return true
		}
	}
	return len(s.ips) == 0
}

// IsDomainInScope checks if a domain is in the scope, but does not check if ip of domain is in scope
func (s *Scope) IsDomainInScope(domain string) bool {
	for _, excluded := range s.excludedDomains {
		if domain == excluded {
			return false
		}
	}
	for _, includedDomain := range s.domains {
		if domain == includedDomain {
			return true
		}
	}
	return len(s.domains) == 0
}

func (s *Scope) IterateIPs() <-chan net.IP {
	ch := make(chan net.IP)
	go func() {
		defer close(ch)
		for _, ipRange := range s.ips {
			for ip := range ipRange.Iterate() {
				if s.IsIPInScope(ip) {
					ch <- ip
				}
			}
		}
	}()
	return ch
}

func (s *Scope) IterateDomains() <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		for _, domain := range s.domains {
			if s.IsDomainInScope(domain) {
				ch <- domain
			}
		}
	}()
	return ch
}

func (s *Scope) IsArtifactInScope(artifact *data.Artifact) bool {
	domain := artifact.GetDomainFromArtifact()
	if domain != "" {
		if s.IsDomainInScope(domain) {
			return true
		}
	}

	ip := artifact.GetIPFromArtifact()
	if ip != nil {
		if s.IsIPInScope(ip) {
			return true
		}
	}

	if ip == nil {
		if s.IsDomainsIPInScope(domain) {
			return true
		}
	}

	log.Printf("Artifact not in scope (ip, domain, artifact): %v, %v, %v", ip, domain, artifact)

	return false
}

func (s *Scope) IsDomainsIPInScope(domain string) bool {
	if domain != "" {
		ip, err := GetDNSCache().Lookup(domain)
		// ignore error case
		if err == nil {
			// check if any of the resolved ip is in scope (and count it as in scope if any ip is in scope (not perfect but good enough for now))
			if s.IsIPInScope(ip) {
				return true
			}
		}
	}
	return false
}
