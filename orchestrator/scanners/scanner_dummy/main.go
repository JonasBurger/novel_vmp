package main

import (
	"log"

	"my.org/novel_vmp/data"
	"my.org/novel_vmp/pkg/slave"
)

var ScannerName = "scanner_dummy"

func main() {
	err := slave.NewServer(work).Start()
	if err != nil {
		log.Fatal(err)
	}
}

func work(artifact *data.Artifact) {
	newArtifact := data.Artifact{
		ArtifactType: data.ArtifactTypeFinding,
		Value:        "Test",
		Location:     artifact.Location,
		Scanner:      ScannerName,
		Severity:     data.SeverityUnknown,
	}
	slave.SendArtifact(newArtifact)
}
