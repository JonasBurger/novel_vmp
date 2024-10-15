package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/spf13/viper"
	"my.org/novel_vmp/data"
	"my.org/novel_vmp/internal/config"
	dockerutils "my.org/novel_vmp/internal/docker_utils"
	ratelimiter "my.org/novel_vmp/internal/rate_limiter"
)

var portStartMutex = sync.Mutex{}
var portStart = 30000

type ScannerInstance struct {
	Mutex             sync.Mutex
	Name              string // scanner name & docker container name & docker container hostname
	dockerImageName   string
	scannerDataInput  <-chan *data.Artifact
	scannerDataOutput chan<- *data.Artifact
	scannerCtrlInput  <-chan data.ScannerInstanceControllMsg
	Busy              bool
	port              int
	wg                *sync.WaitGroup
	rateLimitType     data.RateLimitType
	requestedKey      string
}

func NewScannerInstance(
	name string,
	dockerImageName string,
	scannerDataInput <-chan *data.Artifact,
	scannerDataOutput chan<- *data.Artifact,
	scannerCtrlInput <-chan data.ScannerInstanceControllMsg,
	wg *sync.WaitGroup,
	rateLimitType data.RateLimitType,
	requestedKey string,
) *ScannerInstance {
	return &ScannerInstance{
		Name:              name,
		dockerImageName:   dockerImageName,
		scannerDataInput:  scannerDataInput,
		scannerDataOutput: scannerDataOutput,
		scannerCtrlInput:  scannerCtrlInput,
		port:              getNextPort(),
		Busy:              true,
		wg:                wg,
		rateLimitType:     rateLimitType,
		requestedKey:      requestedKey,
	}
}

func (s *ScannerInstance) Run() {
	defer s.wg.Done()

	id := dockerutils.RunContainer(s.dockerImageName, s.Name, []string{
		fmt.Sprintf("NOVELVMP_MASTER_HOST=%v", "localhost:1323"),
		fmt.Sprintf("NOVELVMP_TEMPLATE_NAME=%v", s.dockerImageName),
		fmt.Sprintf("NOVELVMP_SCANNER_NAME=%v", s.Name),
		fmt.Sprintf("NOVELVMP_SCANNER_PORT=%v", s.port),
		fmt.Sprintf("NOVELVMP_MAX_REQUESTS=%v", viper.GetInt("max_requests")),
		fmt.Sprintf("NOVELVMP_KEY=%v", config.Keymap[s.requestedKey]),
	})
	defer dockerutils.RemoveContainer(id)
	defer dockerutils.StopContainer(id)
	s.waitForRegister()
	s.UnsetBusy()

	for input := range s.scannerDataInput {
		s.SetBusy()
		start := time.Now()
		SendArtifact(*input, s.getHost())
		s.waitForFinishTask()
		ratelimiter.GetInstance().FreeRateLimitAllocation(input, s.rateLimitType)
		s.UnsetBusy()
		writeTimerResultToFile(time.Since(start), s.dockerImageName)
	}
}

func (s *ScannerInstance) waitForRegister() {
	msg := <-s.scannerCtrlInput
	if msg.ScannerInstance != s.Name && msg.ScannerTemplate != s.dockerImageName && msg.ScannerMsg != data.ScannerMsgRegister {
		log.Fatal("s: wrong register msg")
	}
}

func (s *ScannerInstance) waitForFinishTask() {
	msg := <-s.scannerCtrlInput
	if msg.ScannerInstance != s.Name && msg.ScannerTemplate != s.dockerImageName && msg.ScannerMsg != data.ScannerMsgFinishTask {
		log.Fatal("s: wrong finish task msg")
	}
}

func getNextPort() int {
	portStartMutex.Lock()
	defer portStartMutex.Unlock()
	portStart += 1
	return portStart
}

func (s *ScannerInstance) getHost() string {
	return fmt.Sprintf("localhost:%v", s.port)
}

func (s *ScannerInstance) IsBusy() bool {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	return s.Busy
}

func (s *ScannerInstance) SetBusy() {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	if s.Busy {
		log.Fatal("s: Busy already set")
	}
	s.Busy = true
}

func (s *ScannerInstance) UnsetBusy() {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	if !s.Busy {
		log.Fatal("s: Busy already unset")
	}
	s.Busy = false
}

func SendArtifact(artifact data.Artifact, slaveHost string) {

	// send artifact to master
	artifactJson, err := json.Marshal(artifact)
	if err != nil {
		log.Fatal(err)
	}
	msg, err := http.Post("http://"+slaveHost+"/artifact", "application/json", bytes.NewBuffer(artifactJson))
	if err != nil {
		log.Fatal(err)
	}
	if msg.StatusCode != http.StatusOK {
		log.Fatal("Slave did not return 200 OK")
	}
}

func writeTimerResultToFile(duration time.Duration, name string) {
	// Format the current timestamp
	timestamp := time.Now().Format("20060102_150405.000")
	filename := fmt.Sprintf("timings/timer_result_%s_%s.txt", name, timestamp)

	// Create and open the file
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Write the duration to the file
	_, err = file.WriteString(duration.String())
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	fmt.Println("File written successfully:", filename)
}
