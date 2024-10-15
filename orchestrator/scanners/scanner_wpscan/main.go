package main

import (
	"encoding/json"
	"log"
	"os/exec"
	"strconv"

	"my.org/novel_vmp/data"
	"my.org/novel_vmp/pkg/slave"
)

const scannerName string = "scanner_wpscan"

func main() {
	err := slave.NewServer(work).Start()
	if err != nil {
		log.Panic("panic: ", err)
	}
}

func work(artifact *data.Artifact) {
	if artifact.ArtifactType != data.ArtifactTypeTechnology {
		log.Panic("panic: ", scannerName, ": artifact type not supported: ", artifact.ArtifactType)
	}

	// only scan WordPress
	if artifact.Value != "WordPress" {
		return
	}

	url := artifact.Location.URL
	token := slave.Key
	if token == "" {
		log.Panic("panic: ", scannerName, ": missing API token")
	}
	millis_between_requests := int((1.0 / float64(slave.MaxRequests)) * 1000.0)

	cmd := exec.Command("wpscan", "--url", url, "--api-token", token, "--max-threads", "1", "-e", "p,t,cb,dbe", "-f", "json", "--throttle", strconv.Itoa(millis_between_requests))
	jsonData, err := cmd.Output()
	if err != nil {
		if err.Error() != "exit status 5" { // exit status 5 means vulnerabilities found
			log.Printf("wpscan scanned url: %v\n", url)
			log.Printf("wpscan error: %v\n", err)
			log.Printf("wpscan output: %v\n", string(jsonData))
			log.Printf("wpscan command: %v\n", cmd.String())
			log.Printf("millis_between_requests: %v\n", millis_between_requests)
			log.Printf("skipping...\n")
			return // just skip this artifact
			// log.Panic("panic: ", err)
		}
	}

	// Create an instance of the Report struct
	var report Report

	// Parse the JSON data into the struct
	err = json.Unmarshal(jsonData, &report)
	if err != nil {
		log.Printf("wpscan scanned url: %v\n", url)
		log.Printf("wpscan reportd json: %v\n", string(jsonData))
		log.Panic("Error parsing JSON:", err)
	}

	log.Println(report.Banner.Version)

	// wp data
	{
		artifact := data.Artifact{
			ArtifactType: data.ArtifactTypeTechnology,
			Value:        "WordPress",
			Scanner:      scannerName,
			Location:     data.Location{URL: url},
			Version:      report.Version.Number,
			AdditionalData: map[string]interface{}{
				"status":              report.Version.Status,
				"release_date":        report.Version.ReleaseDate,
				"interesting_entries": report.Version.InterestingEntries,
				"found_by":            report.Version.FoundBy,
				"confidence":          report.Version.Confidence,
			},
		}
		slave.SendArtifact(artifact)

		// wp vulnerabilities
		for _, vuln := range report.Version.Vulnerabilities {
			references := []string{}
			if vuln.References != nil {
				for _, refs := range vuln.References {
					if refs != nil {
						references = append(references, refs...)
					}
				}
			}

			artifact := data.Artifact{
				ArtifactType: data.ArtifactTypeFinding,
				Severity:     data.SeverityUnknown,
				Value:        vuln.Title,
				Title:        vuln.Title,
				Scanner:      scannerName,
				Location:     data.Location{URL: url},
				AdditionalData: map[string]interface{}{
					"fixed_in":   vuln.FixedIn,
					"references": references,
				},
			}
			if cve := vuln.tryGetCVE(); cve != nil {
				artifact.CVE = *cve
			}
			slave.SendArtifact(artifact)

		}
	}

	for _, finding := range report.InterestingFindings {
		references := []string{}
		if finding.References.URL != nil {
			for _, ref := range finding.References.URL {
				references = append(references, ref)
			}
		}
		artifact := data.Artifact{
			ArtifactType: data.ArtifactTypeFinding,
			Value:        finding.ToS,
			Scanner:      scannerName,
			Location:     data.Location{URL: finding.URL},
			Severity:     data.SeverityInfo,
			Title:        finding.Type,
			Description:  finding.FoundBy,
			AdditionalData: map[string]interface{}{
				"interesting_entries": finding.InterestingEntries,
				"references":          references,
				"confirmed_by":        finding.ConfirmedBy,
				"confidence":          finding.Confidence,
			},
		}
		slave.SendArtifact(artifact)
	}

	for _, theme := range report.Themes {
		artifact := data.Artifact{
			ArtifactType: data.ArtifactTypeTechnology,
			Value:        "WordPress Theme: " + theme.Slug,
			Version:      theme.VersionDetails.Number,
			Scanner:      scannerName,
			Severity:     data.SeverityInfo,
			Location:     data.Location{URL: theme.Location},
			AdditionalData: map[string]interface{}{
				"latest_version":      theme.LatestVersion,
				"outdated":            theme.Outdated,
				"readme_url":          theme.ReadmeURL,
				"style_name":          theme.StyleName,
				"style_uri":           theme.StyleURI,
				"found_by":            theme.FoundBy,
				"confidence":          theme.Confidence,
				"interesting_entries": theme.InterestingEntries,
				"has_vulnerabilities": len(theme.Vulnerabilities) > 0,
			},
		}
		if len(theme.Vulnerabilities) > 0 {
			artifact.Severity = data.SeverityUnknown
		}
		slave.SendArtifact(artifact)

		// theme vulnerabilities
		for _, vuln := range theme.Vulnerabilities {
			references := []string{}
			if vuln.References != nil {
				for _, refs := range vuln.References {
					if refs != nil {
						references = append(references, refs...)
					}
				}
			}

			artifact := data.Artifact{
				ArtifactType: data.ArtifactTypeFinding,
				Severity:     data.SeverityUnknown,
				Value:        vuln.Title,
				Title:        vuln.Title,
				Scanner:      scannerName,
				Location:     data.Location{URL: url},
				AdditionalData: map[string]interface{}{
					"fixed_in":   vuln.FixedIn,
					"references": references,
				},
			}
			if cve := vuln.tryGetCVE(); cve != nil {
				artifact.CVE = *cve
			}
			slave.SendArtifact(artifact)

		}
	}

	// TODO
	for _, plugin := range report.Plugins {
		artifact := data.Artifact{
			ArtifactType: data.ArtifactTypeTechnology,
			Title:        "WordPress Plugin: " + plugin.Slug,
			Value:        plugin.VersionDetails.Number,
			Scanner:      scannerName,
			Severity:     data.SeverityInfo,
			Location:     data.Location{URL: url},
			AdditionalData: map[string]interface{}{
				"latest_version":      plugin.LatestVersion,
				"outdated":            plugin.Outdated,
				"has_vulnerabilities": len(plugin.Vulnerabilities) > 0,
			},
		}
		if len(plugin.Vulnerabilities) > 0 {
			artifact.Severity = data.SeverityUnknown
		}
		slave.SendArtifact(artifact)

		// theme vulnerabilities
		for _, vuln := range plugin.Vulnerabilities {
			references := []string{}
			if vuln.References != nil {
				for _, refs := range vuln.References {
					if refs != nil {
						references = append(references, refs...)
					}
				}
			}

			artifact := data.Artifact{
				ArtifactType: data.ArtifactTypeFinding,
				Severity:     data.SeverityUnknown,
				Value:        vuln.Title,
				Title:        vuln.Title,
				Scanner:      scannerName,
				Location:     data.Location{URL: url},
				AdditionalData: map[string]interface{}{
					"fixed_in":   vuln.FixedIn,
					"references": references,
				},
			}
			if cve := vuln.tryGetCVE(); cve != nil {
				artifact.CVE = *cve
			}
			slave.SendArtifact(artifact)
		}
	}

	// TODO handle config_backups & db_exports
	if len(report.ConfigBackups) > 0 || len(report.DBExports) > 0 {
		log.Println("WARNING: config_backups & db_exports not implemented")
	}

}

// Main struct
type Report struct {
	Banner                Banner                 `json:"banner"`
	StartTime             int64                  `json:"start_time"`
	StartMemory           int64                  `json:"start_memory"`
	TargetURL             string                 `json:"target_url"`
	TargetIP              string                 `json:"target_ip"`
	EffectiveURL          string                 `json:"effective_url"`
	InterestingFindings   []InterestingFinding   `json:"interesting_findings"`
	Version               Version                `json:"version"`
	MainTheme             Theme                  `json:"main_theme"`
	Plugins               map[string]Plugin      `json:"plugins"`
	Themes                map[string]Theme       `json:"themes"`
	ConfigBackups         map[string]interface{} `json:"config_backups"`
	DBExports             map[string]interface{} `json:"db_exports"`
	VulnAPI               VulnAPI                `json:"vuln_api"`
	StopTime              int64                  `json:"stop_time"`
	Elapsed               int                    `json:"elapsed"`
	RequestsDone          int                    `json:"requests_done"`
	CachedRequests        int                    `json:"cached_requests"`
	DataSent              int64                  `json:"data_sent"`
	DataSentHumanised     string                 `json:"data_sent_humanised"`
	DataReceived          int64                  `json:"data_received"`
	DataReceivedHumanised string                 `json:"data_received_humanised"`
	UsedMemory            int64                  `json:"used_memory"`
	UsedMemoryHumanised   string                 `json:"used_memory_humanised"`
}

// Banner struct
type Banner struct {
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Authors     []string `json:"authors"`
	Sponsor     string   `json:"sponsor"`
}

// InterestingFinding struct
type InterestingFinding struct {
	URL                string                 `json:"url"`
	ToS                string                 `json:"to_s"`
	Type               string                 `json:"type"`
	FoundBy            string                 `json:"found_by"`
	Confidence         int                    `json:"confidence"`
	ConfirmedBy        map[string]interface{} `json:"confirmed_by"`
	References         References             `json:"references"`
	InterestingEntries []string               `json:"interesting_entries"`
}

// References struct
type References struct {
	URL        []string `json:"url"`
	Metasploit []string `json:"metasploit"`
}

// Version struct
type Version struct {
	Number             string                 `json:"number"`
	ReleaseDate        string                 `json:"release_date"`
	Status             string                 `json:"status"`
	FoundBy            string                 `json:"found_by"`
	Confidence         int                    `json:"confidence"`
	InterestingEntries []string               `json:"interesting_entries"`
	ConfirmedBy        map[string]interface{} `json:"confirmed_by"`
	Vulnerabilities    []Vulnerability        `json:"vulnerabilities"`
}

type Vulnerability struct {
	Title      string              `json:"title"`
	FixedIn    string              `json:"fixed_in"`
	References map[string][]string `json:"references"`
}

func (v *Vulnerability) tryGetCVE() *string {
	if refs, ok := v.References["cve"]; ok {
		if len(refs) >= 1 {
			cve := "CVE-" + refs[0]
			return &cve
		}
	}
	return nil
}

// Theme struct
type Theme struct {
	Slug               string                     `json:"slug"`
	Location           string                     `json:"location"`
	LatestVersion      string                     `json:"latest_version"`
	LastUpdated        string                     `json:"last_updated"`
	Outdated           bool                       `json:"outdated"`
	ReadmeURL          interface{}                `json:"readme_url"`
	DirectoryListing   bool                       `json:"directory_listing"`
	ErrorLogURL        *string                    `json:"error_log_url"`
	StyleURL           string                     `json:"style_url"`
	StyleName          string                     `json:"style_name"`
	StyleURI           string                     `json:"style_uri"`
	Description        string                     `json:"description"`
	Author             string                     `json:"author"`
	AuthorURI          string                     `json:"author_uri"`
	Template           *string                    `json:"template"`
	License            string                     `json:"license"`
	LicenseURI         string                     `json:"license_uri"`
	Tags               string                     `json:"tags"`
	TextDomain         string                     `json:"text_domain"`
	FoundBy            string                     `json:"found_by"`
	Confidence         int                        `json:"confidence"`
	InterestingEntries []string                   `json:"interesting_entries"`
	ConfirmedBy        map[string]DetectionMethod `json:"confirmed_by"`
	Vulnerabilities    []Vulnerability            `json:"vulnerabilities"`
	VersionDetails     VersionDetails             `json:"version"`
	Parents            []interface{}              `json:"parents"`
}

// Plugin struct
type Plugin struct {
	Slug            string          `json:"slug"`
	LatestVersion   string          `json:"latest_version"`
	LastUpdated     string          `json:"last_updated"`
	Outdated        bool            `json:"outdated"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	VersionDetails  VersionDetails  `json:"version"`
}

// DetectionMethod struct
type DetectionMethod struct {
	Confidence         int      `json:"confidence"`
	InterestingEntries []string `json:"interesting_entries"`
}

// VersionDetails struct
type VersionDetails struct {
	Number             string                 `json:"number"`
	Confidence         int                    `json:"confidence"`
	FoundBy            string                 `json:"found_by"`
	InterestingEntries []string               `json:"interesting_entries"`
	ConfirmedBy        map[string]interface{} `json:"confirmed_by"`
}

// VulnAPI struct
type VulnAPI struct {
	Plan                   string      `json:"plan"`
	RequestsDoneDuringScan int         `json:"requests_done_during_scan"`
	RequestsRemaining      interface{} `json:"requests_remaining"`
}

// func handleResult(result output.Result) {
// 	if result.Response == nil && result.PassiveReference == nil {
// 		log.Println("Skipping ", result.Request.URL, " because it did not return a response: ", result.Error)
// 		return
// 	}
// 	artifact := data.Artifact{
// 		ArtifactType: data.ArtifactTypeHttpMsg,
// 		Scanner:      scannerName,
// 		Value:        result.Request.URL,
// 		Location: data.Location{
// 			URL: result.Request.URL,
// 		},
// 		Request:  result.Request.Raw,
// 		Response: result.Response.Raw,
// 	}
// 	if result.PassiveReference != nil {
// 		artifact.ResponseJsDom = result.PassiveReference.Source
// 	}
// 	slave.SendArtifact(artifact)
// }
