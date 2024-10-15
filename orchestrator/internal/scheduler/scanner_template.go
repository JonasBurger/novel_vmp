package scheduler

import (
	"fmt"
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
	"my.org/novel_vmp/data"
	ratelimiter "my.org/novel_vmp/internal/rate_limiter"
	"my.org/novel_vmp/utils"
)

// todo teardown pods

// every member function should be thread safe
type ScannerTemplate struct {
	Name               string // scanner & docker image name
	ListenForArtifacts []string
	ListenChannel      <-chan utils.Event[*data.Artifact]
	artifactEventBus   *utils.EventBus[*data.Artifact]
	ArtifactBuffer     ratelimiter.RateLimitedArtifactQueue
	scannerDataInput   chan *data.Artifact
	scannerDataOutput  chan *data.Artifact
	scannerOutputMap   map[*ScannerInstance]chan<- data.ScannerInstanceControllMsg
	nameCounter        int
	instancesWg        sync.WaitGroup
	Config             ScannerConfig
}

func NewScannerTemplate(name string, artifactEventBus *utils.EventBus[*data.Artifact]) *ScannerTemplate {
	config := loadScannerConfig(name)
	inputArtifacts := config.Inputs
	listenChannel := artifactEventBus.Subscribe(inputArtifacts...)

	return &ScannerTemplate{
		Name:               name,
		ListenForArtifacts: inputArtifacts,
		ArtifactBuffer:     *ratelimiter.NewRateLimitedArtifactQueue(config.RateLimitType),
		scannerDataInput:   make(chan *data.Artifact),
		scannerDataOutput:  make(chan *data.Artifact, 100),
		artifactEventBus:   artifactEventBus,
		ListenChannel:      listenChannel,
		scannerOutputMap:   make(map[*ScannerInstance]chan<- data.ScannerInstanceControllMsg),
		Config:             config,
	}
}

type ScannerConfig struct {
	Inputs        []string           `yaml:"inputs"`
	Outputs       []string           `yaml:"outputs"`
	RateLimitType data.RateLimitType `yaml:"rate_limit_type"`
	Instances     int                `yaml:"instances"`
	IgnoreScope   bool               `yaml:"ignore_scope"`
	NeedsKey      string             `yaml:"needs_key"`
}

func loadScannerConfig(scanner string) ScannerConfig {
	path := fmt.Sprintf("scanners/%v/config.yaml", scanner)
	file, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Unable to read config file: %v", err)
	}
	var config ScannerConfig
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		log.Fatalf("Unable to parse config file: %v", err)
	}
	if len(config.Inputs) == 0 {
		log.Fatalf("No inputs defined in config file: %v", path)
	}
	if config.RateLimitType == "" {
		log.Fatalf("No rate_limit_type defined in config file: %v", path)
	}
	if config.RateLimitType != data.RateLimitTypeDisabled && config.RateLimitType != data.RateLimitTypeDomain && config.RateLimitType != data.RateLimitTypeIP {
		log.Fatalf("Invalid rate_limit_type defined in config file: %v", path)
	}
	return config
}

func (s *ScannerTemplate) AddScanner() {
	s.instancesWg.Add(1)
	msgChan := make(chan data.ScannerInstanceControllMsg, 100)
	scanner := NewScannerInstance(s.generateName(), s.Name, s.scannerDataInput, s.scannerDataOutput, msgChan, &s.instancesWg, s.Config.RateLimitType, s.Config.NeedsKey)
	s.scannerOutputMap[scanner] = msgChan
	go scanner.Run()
}

func (s *ScannerTemplate) generateName() (name string) {
	name = fmt.Sprintf("%v_%v", s.Name, s.nameCounter)
	s.nameCounter++
	return
}

func (s *ScannerTemplate) PublishCollectScannerWork() (busy bool) {

	// loop till all input is collected and output is sent
	for {

		// receive events from eventbus
		noInput := false
		select {
		case event := <-s.ListenChannel:
			artifact := event.Payload
			if s.Config.IgnoreScope || CurrentScope.IsArtifactInScope(artifact) {
				s.ArtifactBuffer.Add(artifact)
			}
		default: // would block
			noInput = true
		}

		// send data to scanners instances
		noOutput := true
		if artifact := s.ArtifactBuffer.Pop(); artifact != nil {
			select {
			case s.scannerDataInput <- artifact:
				// scanner instance was waiting for input artifact and has now received one
				noOutput = false
			default: // would block
				// no scanner instance was waiting for input artifact
				s.ArtifactBuffer.ReversePop(artifact)
			}
		}

		// // receive data from pods
		// for _, outputResults := range s.scannerOutputMap {
		// 	select {
		// 	case result := <-outputResults:
		// 		s.artifactEventBus.PublishEvent(utils.Event[*data.Artifact]{Name: "result", Payload: result})
		// 	default: // would block
		// 	}
		// }

		if noInput && noOutput {
			break
		} else {
			busy = true
		}
	}
	return
}

func (s *ScannerTemplate) IsBusy() bool {
	// TODO: possible datarace when scanner already has new work through the channel, but is not set to busy yet

	if s.ArtifactBuffer.Len() > 0 || len(s.ListenChannel) > 0 {
		return true
	}

	for scanner := range s.scannerOutputMap {
		if scanner.IsBusy() {
			return true
		}
	}

	return false
}

func (s *ScannerTemplate) HandleScannerInstanceMsg(msg data.ScannerInstanceControllMsg) {
	instanceName := msg.InstanceName()
	for scanner, msgChan := range s.scannerOutputMap {
		if scanner.Name == instanceName {
			msgChan <- msg
			return
		}
	}
	log.Fatalf("Scanner not found: %v", instanceName)
}

func (s *ScannerTemplate) Close() {
	close(s.scannerDataInput)

	// wait for scanners to finish
	s.instancesWg.Wait()
}

func (s *ScannerTemplate) PrintStatus() {
	log.Printf("ScannerTemplate: %v, instances: %v, busy: %v, input: %v, output: %v", s.Name, len(s.scannerOutputMap), s.IsBusy(), s.ArtifactBuffer.Len(), len(s.ListenChannel))
}
