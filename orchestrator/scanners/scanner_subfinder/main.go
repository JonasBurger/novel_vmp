package main

import (
	"io"
	"log"
	"os"
	"strings"

	"github.com/projectdiscovery/subfinder/v2/pkg/resolve"
	"github.com/projectdiscovery/subfinder/v2/pkg/runner"
	"my.org/novel_vmp/data"
	"my.org/novel_vmp/pkg/slave"
)

const scannerName string = "scanner_subfinder"

var subfinder *runner.Runner

var scannedDomains = make(map[string]struct{})

func main() {
	// fake cmd arguments to get default initialized options
	os.Args = append(os.Args, "--domain")
	os.Args = append(os.Args, "localhost")
	os.Args = append(os.Args, "--all")

	options := runner.ParseOptions()

	options.ResultCallback = handleResult
	options.Domain = []string{}

	var err error
	subfinder, err = runner.NewRunner(options)
	if err != nil {
		log.Panic("Could not create runner: ", err)
	}

	err = slave.NewServer(work).Start()
	if err != nil {
		log.Panic("panic: ", err)
	}

}

func work(artifact *data.Artifact) {

	if artifact.ArtifactType != data.ArtifactTypeDomain {
		log.Panic("panic: ", "scanner_subfinder: artifact type not supported: ", artifact.ArtifactType)
	}

	for domain := range scannedDomains {
		if strings.Contains(artifact.Value, domain) {
			log.Printf("Skipping %s as it was already scanned via %s\n", artifact.Value, domain)
			return
		}
	}

	scannedDomains[artifact.Value] = struct{}{}
	err := subfinder.EnumerateSingleDomain(artifact.Value, []io.Writer{os.Stdout})
	if err != nil {
		log.Panic("Could not run enumeration: ", err)
	}

}

func handleResult(result *resolve.HostEntry) {
	artifact := data.Artifact{
		ArtifactType: data.ArtifactTypeDomain,
		Scanner:      scannerName,
		Value:        result.Host,
		AdditionalData: map[string]interface{}{
			"source": result.Source,
		},
		Location: data.Location{
			URL: "http://" + result.Host + "/",
		},
	}
	slave.SendArtifact(artifact)
}
