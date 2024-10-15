package data

type ArtifactNamed struct {
	Artifact
	ScannerTemplate string `json:"scanner_template" validate:"required"`
	ScannerInstance string `json:"scanner_instance" validate:"required"`
}

const (
	ScannerMsgRegister   = "register"
	ScannerMsgUnregister = "unregister"
	ScannerMsgFinishTask = "finish_task"
)

type ScannerInstanceControllMsg struct {
	ScannerTemplate string `json:"scanner_template" validate:"required"`
	ScannerInstance string `json:"scanner_instance" validate:"required"`
	ScannerMsg      string `json:"scanner_msg" validate:"required"`
}

type ScannerInstanceMsg interface {
	TemplateName() string
	InstanceName() string
}

// ensure interface implementations
var _ ScannerInstanceMsg = (*ScannerInstanceControllMsg)(nil)
var _ ScannerInstanceMsg = (*ArtifactNamed)(nil)

func (s *ScannerInstanceControllMsg) TemplateName() string {
	return s.ScannerTemplate
}

func (a *ArtifactNamed) TemplateName() string {
	return a.ScannerTemplate
}

func (s *ScannerInstanceControllMsg) InstanceName() string {
	return s.ScannerInstance
}

func (a *ArtifactNamed) InstanceName() string {
	return a.ScannerInstance
}
