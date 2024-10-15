package slave

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"my.org/novel_vmp/data"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
)

var MasterHost = os.Getenv("NOVELVMP_MASTER_HOST")
var TemplateName = os.Getenv("NOVELVMP_TEMPLATE_NAME")
var ScannerName = os.Getenv("NOVELVMP_SCANNER_NAME")
var Port = os.Getenv("NOVELVMP_SCANNER_PORT")
var maxRequestsStr = os.Getenv("NOVELVMP_MAX_REQUESTS")
var MaxRequests = (func() int {
	if maxRequestsStr == "" {
		return -1
	}
	reqs, _ := strconv.Atoi(maxRequestsStr)
	return reqs

}())
var Key = os.Getenv("NOVELVMP_KEY")

type Server struct {
	echo          *echo.Echo
	workFuncMutex sync.Mutex
	workFunc      func(*data.Artifact)
}

func NewServer(workFunc func(*data.Artifact)) (s *Server) {
	log.Printf("MasterHost: %v\n", MasterHost)
	log.Printf("TemplateName: %v\n", TemplateName)
	log.Printf("ScannerName: %v\n", ScannerName)
	log.Printf("Port: %v\n", Port)
	log.Printf("MaxRequests: %v\n", MaxRequests)

	s = &Server{
		echo:     echo.New(),
		workFunc: workFunc,
	}
	s.echo.Validator = &CustomValidator{validator: validator.New()}
	s.bindRoutes()
	return
}

func (s *Server) Start() error {
	log.Println("Starting server")
	go func() {
		time.Sleep(100 * time.Millisecond)
		SendMsgRegister()
	}()
	return s.echo.Start(":" + Port)
}

func (s *Server) bindRoutes() {
	s.echo.GET("/status", s.status)
	s.echo.POST("/artifact", s.postArtifact)
}

func (s *Server) status(c echo.Context) error {
	return c.String(http.StatusOK, "running")
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
	artifact := new(data.Artifact)
	if err := c.Bind(artifact); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(artifact); err != nil {
		return err
	}

	log.Println("Received artifact - Client: ", artifact.Title, " - ", artifact.Value)

	go func() {
		s.workFuncMutex.Lock()
		defer s.workFuncMutex.Unlock()
		s.workFunc(artifact)
		log.Println("Finished Task")
		SendMsgFinishTask()
	}()

	return c.String(http.StatusOK, "")
}
