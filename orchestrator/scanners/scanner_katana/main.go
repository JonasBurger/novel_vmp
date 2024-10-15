package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/projectdiscovery/katana/pkg/engine/hybrid"
	"github.com/projectdiscovery/katana/pkg/output"
	"github.com/projectdiscovery/katana/pkg/types"
	"my.org/novel_vmp/data"
	"my.org/novel_vmp/pkg/slave"
)

const scannerName string = "scanner_katana"

func main() {
	err := slave.NewServer(work).Start()
	if err != nil {
		log.Panic("panic: ", err)
	}
}

func work(artifact *data.Artifact) {
	if artifact.ArtifactType != data.ArtifactTypeHost {
		log.Panic("panic: ", scannerName, ": artifact type not supported: ", artifact.ArtifactType)
	}

	// replace 127.* by localhost, because katana does not support ip scopes
	parts := strings.Split(artifact.Value, ":")
	ip_or_domain := parts[0]
	host := artifact.Value
	//println("ip or domain:", ip_or_domain)
	if ip := net.ParseIP(ip_or_domain); ip != nil {
		if ip_or_domain == "127.0.0.1" {
			host = fmt.Sprintf("%s:%s", "localhost", parts[1])
		}
	}

	protocolStr, err := getProtcolString(host)
	if err != nil {
		log.Printf("Host %v does not speak http or https", host)
		return
	}
	url := fmt.Sprintf("%s%s", protocolStr, host)

	if redirect, _ := checkForRedirectToHTTPS(url); redirect {
		log.Printf("Host %v redirected to HTTPS", host)
		// TODO: Maybe send a message to the master that the URL was redirected to HTTPS

		// artifact := data.Artifact{
		// 	ArtifactType: data.ArtifactTypeHttpMsg,
		// 	Scanner:      scannerName,
		// 	Value:        result.Request.URL,
		// 	Location: data.Location{
		// 		URL: result.Request.URL,
		// 	},
		// 	Request: result.Request.Raw,
		// 	AdditionalData: map[string]interface{}{
		// 		"note": "HTTP-Request was aborted by Crawler",
		// 	},
		// }
		// slave.SendArtifact(artifact)
		return
	}

	options := &types.Options{
		MaxDepth:               3,                 // Maximum depth to crawl
		BodyReadSize:           math.MaxInt,       // Maximum response size to read
		Timeout:                30,                // Timeout is the time to wait for request in seconds
		Concurrency:            10,                // Concurrency is the number of concurrent crawling goroutines
		Parallelism:            1,                 // Parallelism is the number of urls processing goroutines
		RateLimit:              slave.MaxRequests, // Maximum requests to send per second
		Strategy:               "depth-first",     // Visit strategy (depth-first, breadth-first)
		OnResult:               handleResult,
		Headless:               true,
		UseInstalledChrome:     true,
		ScrapeJSResponses:      true,
		ScrapeJSLuiceResponses: true,
		FieldScope:             "fqdn",
		CrawlDuration:          5 * time.Minute, // Duration in seconds to crawl target from
		//Delay:              1,                 // Delay is the delay between each crawl requests in seconds
		//Scope:              []string{host},
		//Proxy:              "http://localhost:7000",
	}

	crawlerOptions, err := types.NewCrawlerOptions(options)
	if err != nil {
		log.Panic("panic: ", err)
	}
	defer crawlerOptions.Close()
	crawler, err := hybrid.New(crawlerOptions)
	if err != nil {
		log.Panic("panic: ", err)
	}
	defer crawler.Close()
	err = crawler.Crawl(url)
	if err != nil {
		log.Printf("Could not crawl %s: %s\n", url, err.Error())
		log.Println("Skipping: ", url)
		return
	}

}

func handleResult(result output.Result) {
	// if result.Response == nil && result.PassiveReference == nil {
	// 	log.Println("Skipping ", result.Request.URL, " because it did not return a response: ", result.Error)
	// 	return
	// }
	if result.Response == nil && result.PassiveReference == nil {
		artifact := data.Artifact{
			ArtifactType: data.ArtifactTypeURL,
			Scanner:      scannerName,
			Value:        result.Request.URL,
			Location: data.Location{
				URL: result.Request.URL,
			},
			Request: result.Request.Raw,
			AdditionalData: map[string]interface{}{
				"note": "HTTP-Request was aborted by Crawler",
			},
		}
		slave.SendArtifact(artifact)
	} else {
		artifact := data.Artifact{
			ArtifactType: data.ArtifactTypeHttpMsg,
			Scanner:      scannerName,
			Value:        result.Request.URL,
			Location: data.Location{
				URL: result.Request.URL,
			},
			Request:  result.Request.Raw,
			Response: result.Response.Raw,
		}
		if result.PassiveReference != nil {
			artifact.ResponseJsDom = result.PassiveReference.Source
		}
		slave.SendArtifact(artifact)
	}
}

func getProtcolString(host string) (string, error) {
	urlHttp := fmt.Sprintf("http://%s", host)
	urlHttps := fmt.Sprintf("https://%s", host)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Check HTTP
	resp, err := client.Get(urlHttp)
	if err == nil && resp.StatusCode != http.StatusBadRequest {
		defer resp.Body.Close()
		return "http://", nil
	}

	// Check HTTPS
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	resp, err = client.Get(urlHttps)
	if err == nil && resp.StatusCode != http.StatusBadRequest {
		defer resp.Body.Close()
		return "https://", nil
	}

	return "", fmt.Errorf("unable to determine protocol for %s", host)

}

func checkForRedirectToHTTPS(uri string) (bool, string) {
	domain := getDomainFromURL(uri)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Stop after the first redirect
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(uri)
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()

	// Check if the status code indicates a redirect
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		// Get the redirect location
		redirectURL, err := resp.Location()
		if err != nil {
			return false, ""
		}

		// Parse the redirect URL
		parsedRedirectURL, err := url.Parse(redirectURL.String())
		if err != nil {
			return false, ""
		}

		// Check if the redirect URL is HTTPS and within the same domain
		if parsedRedirectURL.Scheme == "https" && strings.EqualFold(parsedRedirectURL.Hostname(), domain) {
			return true, redirectURL.String()
		}
	}

	return false, ""
}

// getDomainFromURL extracts the domain from a given URL.
func getDomainFromURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		log.Panicf("Error parsing URL: %v", err)
	}

	// Extract the hostname (domain) from the parsed URL.
	domain := parsedURL.Hostname()
	return domain
}
