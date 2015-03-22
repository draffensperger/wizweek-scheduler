package schedule

import (
	"encoding/json"
	"errors"
	"time"
)

func ParseTaskParams(paramsJSON string, taskParams *TaskParams) error {
	return json.Unmarshal([]byte(paramsJSON), taskParams)
}

type TaskParams struct {
	TimeZone          string
	WeeklyTaskBlocks  [][]TimeBlock
	Tasks             []Task
	StartTaskSchedule time.Time
	EndTaskSchedule   time.Time
}

type Task struct {
	Title          string
	EstimatedHours float64
	Reward         float64
	Deadline       time.Time
	StartOnOrAfter time.Time
}

type TimeBlock struct {
	Start TimeWithoutDate
	End   TimeWithoutDate
}

// From: https://gist.github.com/smagch/d2a55c60bbd76930c79f
type TimeWithoutDate struct {
	time.Time
}

type DateWithoutTime struct {
	time.Time
}

const TimeLayout = "15:04"

var TimeParseError = errors.New(`TimeParseError: should be a string formatted as "15:04"`)

func (t TimeWithoutDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.Format(TimeLayout) + `"`), nil
}

func (t *TimeWithoutDate) UnmarshalJSON(b []byte) error {
	s := string(b)
	// len(`"23:59"`) == 7
	if len(s) != 7 {
		return TimeParseError
	}
	ret, err := time.Parse(TimeLayout, s[1:6])
	if err != nil {
		return err
	}
	t.Time = ret
	return nil
}

//func (params *TaskParams) TaskHours(horizonDays int) []time.Time {

//	// Needs to break down into a bunch of different things ...

//	// begin with params.StartTaskSchedule and look at the weekly task blocks
//	// would like to add support for appointments
//	// return times with the specified time zone for the task params
//	// return that list
//	// go until horizonDays
//}
