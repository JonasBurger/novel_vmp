package master

import (
	"net/http"
	"path/filepath"

	"my.org/novel_vmp/data"
	"my.org/novel_vmp/internal/scheduler"

	"log"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

type Server struct {
	scheduler             *scheduler.Scheduler
	echo                  *echo.Echo
	artifactsOutput       chan<- *data.ArtifactNamed
	scannerCtrlMsgsOutput chan<- data.ScannerInstanceControllMsg
}

func NewServer() *Server {
	artifactsChannel := make(chan *data.ArtifactNamed, 10000)
	scannerCtrlMsgs := make(chan data.ScannerInstanceControllMsg, 100)
	s := &Server{
		scheduler:             scheduler.NewScheduler(artifactsChannel, scannerCtrlMsgs),
		echo:                  echo.New(),
		artifactsOutput:       artifactsChannel,
		scannerCtrlMsgsOutput: scannerCtrlMsgs,
	}
	s.echo.Validator = &CustomValidator{validator: validator.New()}
	s.bindRoutes()
	return s
}

func (s *Server) Start() error {

	scanners := viper.GetStringSlice("scanners")
	for _, scannerPath := range scanners {
		scannerName := filepath.Base(scannerPath)
		template := scheduler.NewScannerTemplate(
			scannerName,
			s.scheduler.ArtifactEventBus)
		if !viper.GetBool("scanner-test") {
			for i := 0; i < template.Config.Instances; i++ {
				template.AddScanner()
			}
		}
		s.scheduler.RegisterScannerTemplate(template)
	}

	if !viper.GetBool("scanner-test") {

		go s.scheduler.Run()
	}
	return s.echo.Start(":1323")
}

func (s *Server) bindRoutes() {
	s.echo.GET("/status", s.status)
	s.echo.POST("/register", s.receiveCtrlMsg)
	s.echo.POST("/unregister", s.receiveCtrlMsg)
	s.echo.POST("/finish_task", s.receiveCtrlMsg)
	s.echo.POST("/artifact", s.postArtifact)
}

func (s *Server) status(c echo.Context) error {
	return c.String(http.StatusOK, "running")
}

func (s *Server) receiveCtrlMsg(c echo.Context) error {
	ctrlMsg := data.ScannerInstanceControllMsg{}
	if err := c.Bind(&ctrlMsg); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&ctrlMsg); err != nil {
		return err
	}

	log.Printf("Received ctrl msg from %v: %v", ctrlMsg.ScannerInstance, ctrlMsg.ScannerMsg)

	if !viper.GetBool("scanner-test") {
		s.scannerCtrlMsgsOutput <- ctrlMsg
	} else {
		if ctrlMsg.ScannerMsg == data.ScannerMsgRegister {
			s.scheduler.SendFittingTestArtifact(ctrlMsg.ScannerTemplate, "localhost:10001")
		}
	}

	return c.String(http.StatusOK, "")
}

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func (s *Server) postArtifact(c echo.Context) error {
	artifact := new(data.ArtifactNamed)
	if err := c.Bind(artifact); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(artifact); err != nil {
		return err
	}

	if viper.GetBool("scanner-test") {
		log.Println("Received artifact - Server: ", artifact.Title, " - ", artifact.Value)
	} else {
		if len(s.artifactsOutput) == cap(s.artifactsOutput) {
			log.Printf("Warning: %s Artifacts channel is full\n", artifact.ArtifactType)
		}
		s.artifactsOutput <- artifact
	}

	return c.String(http.StatusOK, "")
}
