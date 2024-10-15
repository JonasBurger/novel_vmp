package slave

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"my.org/novel_vmp/data"
)

func SendArtifact(artifact data.Artifact) {
	checkEnvironment()

	log.Println("Sending Artifact to master: ", artifact.Title, " - ", artifact.Value)

	namedArtifact := data.ArtifactNamed{
		Artifact:        artifact,
		ScannerTemplate: TemplateName,
		ScannerInstance: ScannerName,
	}

	// send artifact to master
	artifactJson, err := json.Marshal(namedArtifact)
	if err != nil {
		log.Panic(err)
	}
	response, err := http.Post("http://"+MasterHost+"/artifact", "application/json", bytes.NewBuffer(artifactJson))
	if err != nil {
		log.Panic(err)
	}
	if response.StatusCode != http.StatusOK {
		log.Panic("Master did not return 200 OK")
	}
}

var registered = false

func SendMsgRegister() {
	checkEnvironment()
	if registered {
		log.Panic("Already registered")
	}
	registered = true
	sendCtrlMsg(data.ScannerMsgRegister)
}

func SendMsgFinishTask() {
	checkEnvironment()
	sendCtrlMsg(data.ScannerMsgFinishTask)
}

func sendCtrlMsg(msg string) {
	ctrlMsg := data.ScannerInstanceControllMsg{
		ScannerTemplate: TemplateName,
		ScannerInstance: ScannerName,
		ScannerMsg:      msg,
	}
	msgJson, err := json.Marshal(ctrlMsg)
	if err != nil {
		log.Panic(err)
	}
	response, err := http.Post("http://"+MasterHost+"/register", "application/json", bytes.NewBuffer(msgJson))
	if err != nil {
		log.Panic(err)
	}
	if response.StatusCode != http.StatusOK {
		log.Panic("Master did not return 200 OK")
	}
}

func checkEnvironment() {
	if MasterHost == "" || TemplateName == "" || ScannerName == "" || Port == "" {
		log.Panic("NOVELVMP_MASTER_HOST, NOVELVMP_TEMPLATE_NAME, NOVELVMP_SCANNER_NAME and NOVELVMP_SCANNER_PORT environment variables must be set")
	}
}
