package storage

import (
	"log"

	"my.org/novel_vmp/data"
	"my.org/novel_vmp/utils"
)

type DeduplacatingStorage struct {
	Artifacts        []*data.Artifact
	artifactEventBus *utils.EventBus[*data.Artifact]
}

func NewDeduplacatingStorage(artifactEventBus *utils.EventBus[*data.Artifact]) *DeduplacatingStorage {
	return &DeduplacatingStorage{
		Artifacts:        []*data.Artifact{},
		artifactEventBus: artifactEventBus,
	}
}

func (d *DeduplacatingStorage) AddArtifact(artifact *data.Artifact) {
	if d.deduplicate(artifact) {
		log.Printf("Depuplicate artifact: %v", artifact)
		return
	}
	d.Artifacts = append(d.Artifacts, artifact)
	d.artifactEventBus.Publish(artifact.ArtifactType, artifact)
}

func (d *DeduplacatingStorage) deduplicate(artifact *data.Artifact) (dropArtifact bool) {
	switch artifact.ArtifactType {
	case data.ArtifactTypeDomain:
	case data.ArtifactTypeIP:
	case data.ArtifactTypeHost:
	case data.ArtifactTypeURL:
		for _, a := range d.Artifacts {
			if a.ArtifactType == artifact.ArtifactType && a.Value == artifact.Value {
				return true
			}
		}
	case data.ArtifactTypeHttpMsg:
		for _, a := range d.Artifacts {
			if a.ArtifactType == artifact.ArtifactType && a.Location.URL == artifact.Location.URL {
				return true
			}
		}
	case data.ArtifactTypeFinding:
		break // no deduplication
	case data.ArtifactTypeTechnology:
		domain := artifact.GetDomainFromArtifact()
		if domain == "" {
			break
		}
		for _, a := range d.Artifacts {
			if a.ArtifactType == data.ArtifactTypeTechnology && a.Value == artifact.Value && a.GetDomainFromArtifact() == domain {
				if a.Version == artifact.Version {
					a.Location.URL = unifyURL(a.Location.URL, artifact.Location.URL)
				}
				if a.Version == "" {
					a.Version = artifact.Version
				}
				return true
			}
		}

	case data.ArtifactTypeScreenshot:
	case data.ArtifactTypeCMS:
		panic("Not implemented")
	}
	return false
}

func unifyURL(urlStr1 string, urlStr2 string) string {
	if urlStr1 == "" {
		return urlStr2
	}
	if urlStr2 == "" {
		return urlStr1
	}
	if urlStr1 == urlStr2 {
		return urlStr1
	}
	return commonPrefix(urlStr1, urlStr2)
}

func commonPrefix(str1, str2 string) string {
	minLength := len(str1)
	if len(str2) < minLength {
		minLength = len(str2)
	}

	prefix := ""
	for i := 0; i < minLength; i++ {
		if str1[i] == str2[i] {
			prefix += string(str1[i])
		} else {
			break
		}
	}

	return prefix
}
