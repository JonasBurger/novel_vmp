package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"my.org/novel_vmp/data"
	"my.org/novel_vmp/pkg/slave"

	nuclei "github.com/projectdiscovery/nuclei/v3/lib"
	"github.com/projectdiscovery/nuclei/v3/pkg/model/types/severity"
	"github.com/projectdiscovery/nuclei/v3/pkg/output"
)

const scannerName string = "scanner_nuclei"

func main() {
	just_update := len(os.Args) > 1 && os.Args[1] == "--update"

	ne, err := nuclei.NewNucleiEngineCtx(
		context.Background(),
		nuclei.WithGlobalRateLimit(slave.MaxRequests, time.Second),
		//nuclei.WithVerbosity(nuclei.VerbosityOptions{Verbose: true}),
		nuclei.WithTemplateUpdateCallback(just_update, func(_newVersion string) {}),
		nuclei.WithTemplateFilters(nuclei.TemplateFilters{
			ExcludeTags: []string{"fuzz"}, // exclude fuzzing & dns templates for now (because they take a long time)
		}),
	)
	if err != nil {
		log.Panic("panic: ", err)
	}
	defer ne.Close()
	// cleanup omitted: ne.Close()

	// exit if called with --update cli argument
	if just_update {
		return
	}

	ne.LoadAllTemplates()

	err = slave.NewServer(work).Start()
	if err != nil {
		log.Panic("panic: ", err)
	}

}

func work(artifact *data.Artifact) {

	ne, err := nuclei.NewNucleiEngineCtx(
		context.Background(),
		nuclei.WithGlobalRateLimit(slave.MaxRequests, time.Second),
		//nuclei.WithVerbosity(nuclei.VerbosityOptions{Verbose: true}),
		nuclei.WithTemplateUpdateCallback(false, func(_newVersion string) {}),
		nuclei.WithTemplateFilters(nuclei.TemplateFilters{
			ExcludeTags: []string{"fuzz"}, // exclude fuzzing & dns templates for now (because they take a long time)
		}),
	)
	if err != nil {
		log.Panic("panic: ", err)
	}
	defer ne.Close()
	// cleanup omitted: ne.Close()

	ne.LoadAllTemplates()

	if artifact.ArtifactType != data.ArtifactTypeDomain && artifact.ArtifactType != data.ArtifactTypeIP {
		log.Panic("panic: ", "scanner_nuclei: artifact type not supported: ", artifact.ArtifactType)
	}

	ne.LoadTargets([]string{artifact.Value}, true)
	err = ne.ExecuteWithCallback(handleResult)
	if err != nil {
		log.Panic("panic: ", err)
	}
}

func handleResult(result *output.ResultEvent) {
	finding := data.Artifact{
		ArtifactType: data.ArtifactTypeFinding,
		Scanner:      scannerName,
		Location: data.Location{
			IP:  result.IP,
			URL: result.URL,
		},
		Value:       result.Matched,
		Severity:    mapSeverity(result.Info.SeverityHolder.Severity),
		Title:       formatTitle(result.Info.Name, result.MatcherName),
		Description: result.Info.Description,
		AdditionalData: map[string]interface{}{
			"template_id": result.TemplateID,
			"tags":        result.Info.Tags.Value,
		},
	}
	if result.Info.Classification != nil {
		finding.CVE = result.Info.Classification.CVEID.String()
		finding.CVSSMetrics = result.Info.Classification.CVSSMetrics
		finding.CVSSScore = result.Info.Classification.CVSSScore
	}
	if result.Info.Reference != nil {
		finding.AdditionalData["references"] = result.Info.Reference.StringSlice.Value
	}
	slave.SendArtifact(finding)
}

func formatTitle(name string, matcherName string) string {
	if matcherName == "" {
		return name
	}
	return fmt.Sprintf("%v: %v", name, matcherName)
}

func mapSeverity(nucleiSeverity severity.Severity) data.Severity {
	switch nucleiSeverity {
	case severity.Info:
		return data.SeverityInfo
	case severity.Low:
		return data.SeverityLow
	case severity.Medium:
		return data.SeverityMedium
	case severity.High:
		return data.SeverityHigh
	case severity.Critical:
		return data.SeverityCritical
	default:
		return data.SeverityUnknown
	}
}
