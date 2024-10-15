package scheduler

const (
	ControllEventStart        = "start"
	ControllEventStop         = "stop"
	ControllEventRegistered   = "registered"
	ControllEventUnregistered = "unregistered"
)

type SchedulerEvent struct {
	Type string `json:"type"`
	Name string `json:"name"`
}
