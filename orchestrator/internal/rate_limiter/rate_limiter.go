package ratelimiter

import (
	"os"
	"strings"
	"sync"

	"log"

	"gopkg.in/yaml.v2"
	"my.org/novel_vmp/data"
)

type RateLimiter struct {
	domains              map[string]struct{}
	ips                  map[string]struct{}
	vserver              map[string]string
	domainVserverMapping map[string]string
	mutex                sync.Mutex
}

var instance *RateLimiter
var once sync.Once

func GetInstance() *RateLimiter {
	once.Do(func() {
		instance = newRateLimiter()
	})
	return instance
}

func newRateLimiter() *RateLimiter {
	domainVserverMapping := make(map[string]string)
	data, err := os.ReadFile("vserver-mapping.yaml")
	if err != nil {
		log.Fatalf("Error reading YAML file: %v", err)
	}

	err = yaml.Unmarshal(data, &domainVserverMapping)
	if err != nil {
		log.Fatalf("Error parsing YAML file: %v", err)
	}
	// remove leading "*." from domains
	for domain, vserver := range domainVserverMapping {
		if domain[0:2] == "*." {
			domainVserverMapping[domain[2:]] = vserver
		}
	}

	return &RateLimiter{
		domains:              make(map[string]struct{}),
		ips:                  make(map[string]struct{}),
		vserver:              make(map[string]string),
		domainVserverMapping: domainVserverMapping,
	}
}

func (r *RateLimiter) getVserverForDomain(domain string) (string, bool) {
	//r.mutex.Lock()
	//defer r.mutex.Unlock()
	for d, v := range r.domainVserverMapping {
		if d == domain {
			return v, true
		}
		if d[0:2] == "*." {
			if strings.Contains(domain, d[2:]) {
				return v, true
			}
		}
	}
	return "", false
}

func (r *RateLimiter) isVserverForDomainInUse(domain string) bool {
	//r.mutex.Lock()
	//defer r.mutex.Unlock()
	vserver, ok := r.getVserverForDomain(domain)
	if !ok {
		// ignore if no vserver mapping is found
		return false
	}
	for _, v := range r.vserver {
		if v == vserver {
			return true
		}
	}
	return false
}

func (r *RateLimiter) setVserverInUse(domain string) {
	//r.mutex.Lock()
	//defer r.mutex.Unlock()
	vserver, ok := r.getVserverForDomain(domain)
	if !ok {
		// ignore if no vserver mapping is found
		return
	}

	for _, v := range r.vserver {
		if v == vserver {
			log.Panicf("Tried to set vserver in use that was already in use: %v", vserver)
		}
	}
	r.vserver[vserver] = domain
}

func (r *RateLimiter) IsDomainInUse(domain string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	_, ok := r.domains[domain]
	if ok {
		return ok
	}
	// Hack: check vserver and domain together
	ok = r.isVserverForDomainInUse(domain)
	return ok
}

func (r *RateLimiter) IsIPInUse(ip string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	_, ok := r.ips[ip]
	return ok
}

func (r *RateLimiter) SetDomainInUse(domain string) (success bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if domain == "" {
		log.Panicf("Tried to set empty domain in use")
	}

	_, ok := r.domains[domain]
	if ok {
		return false
	}

	//log.Printf("!!! Setting domain in use: %v\n", domain)

	r.domains[domain] = struct{}{}
	r.setVserverInUse(domain)
	return true
}

func (r *RateLimiter) SetIPInUse(ip string) (success bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, ok := r.ips[ip]
	if ok {
		return false
	}

	r.ips[ip] = struct{}{}
	return true
}

func (r *RateLimiter) ReleaseDomain(domain string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, ok := r.domains[domain]; !ok {
		log.Printf("Warning: Tried to release domain that was not in use: %v", domain)
		return
	}

	delete(r.domains, domain)
	delete(r.vserver, domain)
}

func (r *RateLimiter) ReleaseIP(ip string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, ok := r.ips[ip]; !ok {
		log.Printf("Warning: Tried to release IP that was not in use: %v", ip)
		return
	}

	delete(r.ips, ip)
}

func IsRateLimitedArtifact(artifact *data.Artifact) bool {
	return artifact.ArtifactType == data.ArtifactTypeHost || artifact.ArtifactType == data.ArtifactTypeDomain || artifact.ArtifactType == data.ArtifactTypeIP || artifact.ArtifactType == data.ArtifactTypeTechnology
}

func (r *RateLimiter) FreeRateLimitAllocation(artifact *data.Artifact, rateLimitType data.RateLimitType) {
	if rateLimitType == data.RateLimitTypeDisabled {
		return
	}

	if IsRateLimitedArtifact(artifact) {
		if rateLimitType == data.RateLimitTypeDomain {
			domain := artifact.GetDomainFromArtifact()
			if domain != "" {
				r.ReleaseDomain(domain)
				return
			}
		}
		ip := artifact.GetIPFromArtifact()
		if ip != nil {
			r.ReleaseIP(ip.String())
		}
	}
}

func (r *RateLimiter) PrintStatus() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	log.Printf("RateLimiter: domains: %v, ips: %v, vserver: %v\n", r.domains, r.ips, r.vserver)
}
