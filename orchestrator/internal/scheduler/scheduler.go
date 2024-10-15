package scheduler

import (
	"log"
	"os"
	"slices"
	"time"

	"my.org/novel_vmp/data"
	ratelimiter "my.org/novel_vmp/internal/rate_limiter"
	"my.org/novel_vmp/internal/results"
	"my.org/novel_vmp/internal/storage"
	"my.org/novel_vmp/utils"
)

type Scheduler struct {
	ArtifactEventBus *utils.EventBus[*data.Artifact]
	ArtifactStorage  *storage.DeduplacatingStorage
	ControllEventBus *utils.EventBus[*SchedulerEvent]
	scannerTemplates []*ScannerTemplate
	artifactsInput   <-chan *data.ArtifactNamed
	scannerCtrlMsgs  <-chan data.ScannerInstanceControllMsg
	artifactDeriver  *DeriveDomainHostFromIPHost
}

func NewScheduler(artifactsInput <-chan *data.ArtifactNamed, scannerCtrlMsgs <-chan data.ScannerInstanceControllMsg) *Scheduler {
	artifactEventBus := utils.NewEventBus[*data.Artifact]()
	return &Scheduler{
		ArtifactEventBus: artifactEventBus,
		ArtifactStorage:  storage.NewDeduplacatingStorage(artifactEventBus),
		ControllEventBus: utils.NewEventBus[*SchedulerEvent](),
		artifactsInput:   artifactsInput,
		scannerCtrlMsgs:  scannerCtrlMsgs,
		artifactDeriver:  NewArtifactDerivations(artifactEventBus),
	}
}

func (s *Scheduler) Run() {
	go s.artifactDeriver.Run()
	s.deriveArtifactsFromScope()

	status_counter := 0

	for {

		busy := false
		should_sleep := true

		for _, st := range s.scannerTemplates {
			doneWork := st.PublishCollectScannerWork()
			if doneWork {
				busy = true
			}
		}

		if artifacts := len(s.artifactsInput); artifacts > 0 {
			busy = true
			for i := 0; i < artifacts; i++ {
				artifact := <-s.artifactsInput
				s.handleRecievedArtifact(artifact)
			}
			should_sleep = false
		}

		if len(s.scannerCtrlMsgs) > 0 {
			busy = true
			msg := <-s.scannerCtrlMsgs
			templateIndex := slices.IndexFunc(s.scannerTemplates, func(p *ScannerTemplate) bool {
				return p.Name == msg.TemplateName()
			})
			if templateIndex == -1 {
				log.Fatalf("Template not found: %v", msg.ScannerTemplate)
			}
			template := s.scannerTemplates[templateIndex]
			template.HandleScannerInstanceMsg(msg)
			should_sleep = false
		}

		if !busy {
			busy = s.IsBusy()
		}

		status_counter++
		if status_counter > 10000 {
			status_counter = 0
			for _, p := range s.scannerTemplates {
				p.PrintStatus()
			}
			ratelimiter.GetInstance().PrintStatus()
		}

		if busy {
			if should_sleep {
				time.Sleep(100 * time.Millisecond)
			}
		} else {
			results.UploadArtifactsToElastic(s.ArtifactStorage.Artifacts)
			log.Println("Finished!")
			s.Close()
			os.Exit(0)
		}
	}

}

func (s *Scheduler) IsBusy() bool {
	if s.ControllEventBus.AreEventsInBus() || s.ArtifactEventBus.AreEventsInBus() {
		return true
	}

	for _, p := range s.scannerTemplates {
		if p.IsBusy() {
			return true
		}
	}
	return false
}

func (s *Scheduler) RegisterScannerTemplate(p *ScannerTemplate) {
	s.scannerTemplates = append(s.scannerTemplates, p)
}

func (s *Scheduler) PublishArtifact(artifact *data.Artifact) {
	s.ArtifactStorage.AddArtifact(artifact)
}

func (s *Scheduler) handleRecievedArtifact(artifact *data.ArtifactNamed) {
	// todo duplicate check
	s.PublishArtifact(&artifact.Artifact)
}

func (s *Scheduler) Close() {
	for _, p := range s.scannerTemplates {
		p.Close()
	}
}

func (s *Scheduler) SendFittingTestArtifact(scannerTemplateName string, host string) {
	var inputArtifactType string
	for _, p := range s.scannerTemplates {
		if p.Name == scannerTemplateName {
			inputArtifactType = p.ListenForArtifacts[0]
			break
		}
	}
	if inputArtifactType == "" {
		log.Fatalf("Scanner template not found: %v", scannerTemplateName)
	}
	artifact := getTestArtifact(inputArtifactType)
	SendArtifact(*artifact, host)
}

func (s *Scheduler) deriveArtifactsFromScope() {
	scope := NewScopeFromViperConfig()

	for ip := range scope.IterateIPs() {
		s.PublishArtifact(&data.Artifact{
			ArtifactType: data.ArtifactTypeIP,
			Value:        ip.String(),
			Scanner:      "scope",
		})
	}

	for domain := range scope.IterateDomains() {
		s.PublishArtifact(&data.Artifact{
			ArtifactType: data.ArtifactTypeDomain,
			Value:        domain,
			Scanner:      "scope",
		})
	}
}
