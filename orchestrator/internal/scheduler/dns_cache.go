package scheduler

import (
	"encoding/gob"
	"net"
	"os"
	"sync"
	"time"
)

type DNSCache struct {
	mu      sync.Mutex
	Cache   map[string]net.IP
	Expires map[string]time.Time
}

var dnsCache *DNSCache

func GetDNSCache() *DNSCache {
	if dnsCache == nil {
		dnsCache = newDNSCache()
	}
	return dnsCache
}

func newDNSCache() *DNSCache {
	d := &DNSCache{
		Cache:   make(map[string]net.IP),
		Expires: make(map[string]time.Time),
	}
	d.loadCacheFromDisk()
	return d
}

func (d *DNSCache) Lookup(domain string) (net.IP, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if domain is in cache and not expired
	if ip, found := d.Cache[domain]; found {
		if time.Now().Before(d.Expires[domain]) {
			return ip, nil
		}
		// Remove expired entry
		delete(d.Cache, domain)
		delete(d.Expires, domain)
	}

	// Perform DNS lookup
	ips, err := net.LookupIP(domain)
	if err != nil {
		return nil, err
	}

	if len(ips) == 0 {
		return nil, net.UnknownNetworkError("no IP addresses found")
	}

	// Cache the result with a TTL of 1 week
	ip := ips[0]
	d.Cache[domain] = ip
	d.Expires[domain] = time.Now().Add(24 * 7 * time.Hour)
	d.saveCacheToDisk()

	return ip, nil
}

func (d *DNSCache) saveCacheToDisk() error {
	file, err := os.Create("dns_cache.gob")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(d)
}

func (d *DNSCache) loadCacheFromDisk() error {
	file, err := os.Open("dns_cache.gob")
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	return decoder.Decode(d)
}
