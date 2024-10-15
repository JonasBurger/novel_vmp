package ratelimiter

import (
	"log"

	"my.org/novel_vmp/data"
)

type RateLimitedArtifactQueue struct {
	artifacts     []*data.Artifact
	rateLimitType data.RateLimitType
}

func NewRateLimitedArtifactQueue(rateLimitType data.RateLimitType) *RateLimitedArtifactQueue {
	return &RateLimitedArtifactQueue{
		artifacts:     []*data.Artifact{},
		rateLimitType: rateLimitType,
	}
}

func (q *RateLimitedArtifactQueue) Add(artifact *data.Artifact) {
	q.artifacts = append(q.artifacts, artifact)
}

func (q *RateLimitedArtifactQueue) ReversePop(artifact *data.Artifact) {
	q.release(artifact)
	q.artifacts = append([]*data.Artifact{artifact}, q.artifacts...)
}

func (q *RateLimitedArtifactQueue) release(artifact *data.Artifact) {
	if q.rateLimitType == data.RateLimitTypeDisabled {
		return
	}
	rateLimiter := GetInstance()
	if q.rateLimitType == data.RateLimitTypeDomain {
		domain := artifact.GetDomainFromArtifact()
		if domain != "" {
			rateLimiter.ReleaseDomain(domain)
			return
		}
	}

	ip := artifact.GetIPFromArtifact()
	if ip != nil {
		rateLimiter.ReleaseIP(ip.String())
	} else {
		log.Printf("RateLimitedArtifactQueue release: failed to extract domain/ip from artifact: %v", artifact)
	}

}

func (q *RateLimitedArtifactQueue) Pop() *data.Artifact {
	if len(q.artifacts) == 0 {
		return nil
	}
	if q.rateLimitType == data.RateLimitTypeDisabled {
		artifact := q.artifacts[0]
		q.artifacts = q.artifacts[1:]
		return artifact
	} else {
		rateLimiter := GetInstance()

		var artifact *data.Artifact
		artifact, q.artifacts = siphonIf(q.artifacts, func(artifact *data.Artifact) bool {
			if q.rateLimitType == data.RateLimitTypeDomain {
				domain := artifact.GetDomainFromArtifact()
				if domain != "" {
					return rateLimiter.SetDomainInUse(domain)
				}
			}
			ip := artifact.GetIPFromArtifact()
			if ip == nil {
				log.Printf("failed to extract domain/ip from artifact: %v", artifact)
				return true
			}
			return rateLimiter.SetIPInUse(ip.String())
		})

		return artifact
	}
}

func (q *RateLimitedArtifactQueue) Len() int {
	return len(q.artifacts)
}

func siphonIf(artifacts []*data.Artifact, f func(artifact *data.Artifact) bool) (*data.Artifact, []*data.Artifact) {
	for i, artifact := range artifacts {
		if f(artifact) {
			return artifact, append(artifacts[:i], artifacts[i+1:]...)
		}
	}
	return nil, artifacts
}
