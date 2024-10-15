package data

import (
	"log"
	"net"
	"net/url"
	"regexp"
)

const (
	ArtifactTypeDomain     = "domain"
	ArtifactTypeIP         = "ip"
	ArtifactTypeHost       = "host" // ip/domain:port
	ArtifactTypeURL        = "url"
	ArtifactTypeCMS        = "cms"
	ArtifactTypeHttpMsg    = "httpmsg" // request/response pair
	ArtifactTypeScreenshot = "screenshot"
	ArtifactTypeFinding    = "finding"
	ArtifactTypeTechnology = "technology"
)

type Artifact struct {
	ArtifactType   string                 `json:"type" validate:"required"`    // like domain
	Value          string                 `json:"value,omitempty"`             // like google.com
	Location       Location               `json:"location"`                    // url/ip of site
	Scanner        string                 `json:"scanner" validate:"required"` // like nuclei
	Severity       Severity               `json:"severity,omitempty"`          // like high
	Title          string                 `json:"title,omitempty"`             // like "XSS..."
	Description    string                 `json:"description,omitempty"`       // optional description
	CVE            string                 `json:"cve,omitempty"`               // optional CVE
	CVSSMetrics    string                 `json:"cvss,omitempty"`              // optional CVSS
	CVSSScore      float64                `json:"cvss_score,omitempty"`        // optional CVSS score
	Data           []byte                 `json:"data,omitempty"`              // optional data (e.g. http body)
	Request        string                 `json:"request,omitempty"`           // optional request
	Response       string                 `json:"response,omitempty"`          // optional response
	ResponseJsDom  string                 `json:"response_dom,omitempty"`      // optional response dom
	AdditionalData map[string]interface{} `json:"additional_data,omitempty"`   // optional additional data
	Version        string                 `json:"version,omitempty"`           // optional version
	Categories     []string               `json:"categories,omitempty"`        // optional categories
}

func (artifact *Artifact) GetIPFromArtifact() net.IP {
	if artifact.Location.IP != "" {
		return net.ParseIP(artifact.Location.IP)
	}
	if artifact.Location.URL != "" {
		if ip := TryGetIpFromURL(artifact.Location.URL); ip != nil {
			return ip
		}
	}

	if artifact.ArtifactType == ArtifactTypeIP {
		return net.ParseIP(artifact.Value)
	}
	if artifact.ArtifactType == ArtifactTypeHost {
		// parse ip from sth like 127.0.0.1:80
		host, _, err := net.SplitHostPort(artifact.Value)
		if err != nil {
			log.Panicf("Error parsing host: %v", err)
		}
		ip := net.ParseIP(host)
		if ip != nil {
			return ip
		}
		// not an ip

	}
	if artifact.ArtifactType == ArtifactTypeURL {
		// parse ip from sth like http://...
		uri, err := url.Parse(artifact.Value)
		if err != nil {
			log.Panicf("Error parsing URL: %v", err)
		}
		hostname := uri.Hostname()
		ip := net.ParseIP(hostname)
		if ip != nil {
			return ip
		}
		// not an ip
	}

	return nil
}

func TryGetIpFromURL(urlStr string) net.IP {
	uri, err := url.Parse(urlStr)
	if err != nil {
		log.Panicf("Error parsing URL: %v", err)
	}
	hostname := uri.Hostname()
	ip := net.ParseIP(hostname)
	return ip
}

func TryGetDomainFromURL(urlStr string) string {
	uri, err := url.Parse(urlStr)
	if err != nil {
		re := regexp.MustCompile(`^(?:https?:\/\/)?([^\/\?]+)`)
		matches := re.FindStringSubmatch(urlStr)
		if len(matches) > 1 {
			return matches[1]
		} else {
			log.Panicf("Error parsing URL: %v", err)
			return ""
		}
	}
	hostname := uri.Hostname()
	return hostname
}

func (artifact *Artifact) GetDomainFromArtifact() string {
	if artifact.ArtifactType == ArtifactTypeDomain {
		return artifact.Value
	}

	if artifact.ArtifactType == ArtifactTypeHost {
		// parse domain from sth like google.com:80
		host, _, err := net.SplitHostPort(artifact.Value)
		if err != nil {
			log.Panicf("Error parsing host: %v", err)
		}
		if net.ParseIP(host) == nil {
			return host
		}
	}

	if artifact.Location.URL != "" {
		domain := TryGetDomainFromURL(artifact.Location.URL)
		return domain
	}
	return ""
}
