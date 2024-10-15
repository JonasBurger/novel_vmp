package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Ullaakut/nmap/v3"
	"my.org/novel_vmp/data"
	"my.org/novel_vmp/pkg/slave"
)

const scannerName string = "scanner_nmap"

func main() {

	err := slave.NewServer(work).Start()
	if err != nil {
		log.Panic("panic: ", err)
	}

}

func work(artifact *data.Artifact) {

	if artifact.ArtifactType != data.ArtifactTypeIP {
		log.Panic("panic: ", "scanner_nmap: artifact type not supported: ", artifact.ArtifactType)
	}

	ctx := context.TODO()

	scanner, err := nmap.NewScanner(
		ctx,
		nmap.WithTargets(artifact.Value),
		nmap.WithMostCommonPorts(1000),
		nmap.WithVerbosity(3),
		nmap.WithMaxRate(slave.MaxRequests),
	)
	if err != nil {
		log.Fatalf("unable to create nmap scanner: %v", err)
	}

	result, warnings, err := scanner.Run()
	if len(*warnings) > 0 {
		log.Printf("run finished with warnings: %s\n", *warnings) // Warnings are non-critical errors from nmap.
	}
	if err != nil {
		log.Fatalf("unable to run nmap scan: %v", err)
	}

	// Use the results to print an example output
	for _, host := range result.Hosts {
		if len(host.Ports) == 0 || len(host.Addresses) == 0 {
			continue
		}
		for _, port := range host.Ports {
			if port.State.State == "open" {
				artifact := data.Artifact{
					ArtifactType: data.ArtifactTypeHost,
					Scanner:      scannerName,
					Location:     data.Location{IP: host.Addresses[0].Addr},
					Value:        fmt.Sprintf("%v:%v", host.Addresses[0].Addr, port.ID),
				}
				slave.SendArtifact(artifact)
			}
		}
	}

	fmt.Printf("Nmap done: %d hosts up scanned in %.2f seconds\n", len(result.Hosts), result.Stats.Finished.Elapsed)
}
