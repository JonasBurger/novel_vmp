package scheduler

import (
	"log"
	"net"
	"strconv"
	"time"

	"my.org/novel_vmp/data"
	"my.org/novel_vmp/utils"
)

type DeriveDomainHostFromIPHost struct {
	artifactEventBus  *utils.EventBus[*data.Artifact]
	listenChannel     <-chan utils.Event[*data.Artifact]
	alreadyKnownHosts map[string]struct{}
	knownDomainsPerIP map[string][]string
	knownPortsPerIP   map[string][]int
}

func NewArtifactDerivations(artifactEventBus *utils.EventBus[*data.Artifact]) *DeriveDomainHostFromIPHost {
	listenChannel := artifactEventBus.Subscribe(data.ArtifactTypeHost, data.ArtifactTypeDomain)

	return &DeriveDomainHostFromIPHost{
		artifactEventBus:  artifactEventBus,
		listenChannel:     listenChannel,
		alreadyKnownHosts: map[string]struct{}{},
		knownDomainsPerIP: map[string][]string{},
		knownPortsPerIP:   map[string][]int{},
	}
}

func (d *DeriveDomainHostFromIPHost) Run() {
	for {

	loop:
		for {
			select {
			case event := <-d.listenChannel:
				d.handleRecievedArtifact(event.Payload)

			default:
				// would block
				break loop
			}

		}

		time.Sleep(100 * time.Millisecond)
	}

}

func (d *DeriveDomainHostFromIPHost) handleRecievedArtifact(artifact *data.Artifact) {
	switch artifact.ArtifactType {
	case data.ArtifactTypeHost:
		d.addHost(artifact.Value)

	case data.ArtifactTypeDomain:
		d.addDomain(artifact.Value)

	case data.ArtifactTypeIP:
		d.addIP(artifact.Value)
	default:
		log.Fatalf("Unknown artifact type: %v", artifact.ArtifactType)

	}
}

func (d *DeriveDomainHostFromIPHost) addHost(host string) {
	d.alreadyKnownHosts[host] = struct{}{} // must come from somewhere, so it is known
	ipOrDomain, port, err := net.SplitHostPort(host)
	if err != nil {
		log.Panicf("Error splitting host and port: %v", err)
	}
	d.addDomainOrIP(ipOrDomain)

	ip := net.ParseIP(ipOrDomain)
	if ip == nil {
		// ipOrDomain is a domain
		ip, err := GetDNSCache().Lookup(ipOrDomain)
		if err != nil {
			log.Panicf("Error looking up domain: %v", err)
		}
		d.addIP(ip.String())
	}

	var portInt int
	portInt, err = strconv.Atoi(port)
	if err != nil {
		log.Panicf("Error converting port to int: %v", err)
	}
	d.addPort(ip.String(), portInt)
}

func (d *DeriveDomainHostFromIPHost) addPort(ip string, port int) {
	_, exists := d.knownPortsPerIP[ip]
	if !exists {
		d.knownPortsPerIP[ip] = []int{}
	}
	d.knownPortsPerIP[ip] = append(d.knownPortsPerIP[ip], port)
	d.generateNewArtifacts(ip)
}

func (d *DeriveDomainHostFromIPHost) addDomainOrIP(domainOrIP string) {
	ip := net.ParseIP(domainOrIP)
	if ip != nil {
		d.addIP(domainOrIP)
	} else {
		d.addDomain(domainOrIP)
	}
}

func (d *DeriveDomainHostFromIPHost) addIP(ip string) {
	_, exists := d.knownPortsPerIP[ip]
	if !exists {
		d.knownPortsPerIP[ip] = []int{}
	}
	d.generateNewArtifacts(ip)
}

func (d *DeriveDomainHostFromIPHost) addDomain(domain string) {
	ip, err := d.getIPForDomain(domain)
	if err != nil {
		return
	}
	d.generateNewArtifacts(ip)
}

func (d *DeriveDomainHostFromIPHost) generateNewArtifacts(ip string) {
	for _, port := range d.knownPortsPerIP[ip] {
		for _, domain := range d.knownDomainsPerIP[ip] {
			hostStr := net.JoinHostPort(domain, strconv.Itoa(port))
			if _, exists := d.alreadyKnownHosts[hostStr]; !exists {
				artifact := &data.Artifact{
					ArtifactType: data.ArtifactTypeHost,
					Value:        hostStr,
					Location: data.Location{
						IP:  ip,
						URL: "",
					},
					Scanner: "Domain:Port derived from IP:Port",
				}
				d.alreadyKnownHosts[hostStr] = struct{}{}
				d.artifactEventBus.Publish(artifact.ArtifactType, artifact)

			}

		}
	}
}

func (d *DeriveDomainHostFromIPHost) getIPForDomain(domain string) (string, error) {
	// search d.knownDomainsPerIP & lookup if not found
	for ip, knownDomains := range d.knownDomainsPerIP {
		for _, knownDomain := range knownDomains {
			if knownDomain == domain {
				return ip, nil
			}
		}
	}
	ip, err := GetDNSCache().Lookup(domain)
	if err != nil {
		log.Printf("Error looking up domain: %v\n", err)
		return "", err
	}
	ipStr := ip.String()
	domains, exists := d.knownDomainsPerIP[ipStr]
	if !exists {
		d.knownDomainsPerIP[ipStr] = []string{domain}
	} else {
		d.knownDomainsPerIP[ipStr] = append(domains, domain)
	}
	return ipStr, nil
}
