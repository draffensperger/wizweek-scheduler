package schedule

import (
	"encoding/json"
	"errors"
	"github.com/k0kubun/pp"
	. "time"
)

func p(x interface{}) {
	pp.Println(x)
}

func ParseTaskParams(paramsJSON string, params *TaskParams) error {
	if err := json.Unmarshal([]byte(paramsJSON), params); err != nil {
		return err
	}

	loc, err := LoadLocation(params.TimeZoneName)
	if err != nil {
		return err
	}
	params.Location = loc
	params.localizeTimes()
	return nil
}

func (tp *TaskParams) localizeTimes() {
	tp.StartTaskSchedule = tp.StartTaskSchedule.In(tp.Location)
	tp.EndTaskSchedule = tp.EndTaskSchedule.In(tp.Location)

	for i := 0; i < len(tp.Tasks); i++ {
		tp.Tasks[i].Deadline = tp.Tasks[i].Deadline.In(tp.Location)
		tp.Tasks[i].StartOnOrAfter = tp.Tasks[i].StartOnOrAfter.In(tp.Location)
	}
}

type TaskParams struct {
	TimeZoneName string `json:"timeZone"`
	*Location
	WeeklyTaskBlocks  [][]TimeBlock
	Tasks             []Task
	StartTaskSchedule Time
	EndTaskSchedule   Time
}

type Task struct {
	Title          string
	EstimatedHours float64
	Reward         float64
	Deadline       Time
	StartOnOrAfter Time
}

type TimeBlock struct {
	Start TimeWithoutDate
	End   TimeWithoutDate
}

// From: https://gist.github.com/smagch/d2a55c60bbd76930c79f
type TimeWithoutDate struct {
	Time
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
	ret, err := Parse(TimeLayout, s[1:6])
	if err != nil {
		return err
	}
	t.Time = ret
	return nil
}

//func (tp *TaskParams) UnmarshalJSON(b []byte) error {
//	tpAutoUnmarshal := taskParamsAutoUnmarshal{}
//	if json.Unmarshal(b, &tpAutoUnmarshal); err == nil {
//		tp
//	}
//}

func (params *TaskParams) TaskHours() []Time {
	/*
		General strategy:
		1. Start at the start of the first time block. Move forward hour by hour
		skip hours if there is an appointment during that hour.
		2. When finished with the first time block, move to the next one
		3. When done with the last time block, move back to the first but the next week
		4. Return the list of available hours
	*/
	//	tz = params.TimeZone.Location

	//	blockStartEndTimes = make([][2]Time)
	//	weekStart = params.StartTaskSchedule.Weekday()
	//	blocks = params.WeeklyTaskBlocks
	//	block = [params.StartTaskSchedule.Weekday()]

	//	hours := make([]Time)
	//	cur := params.StartTaskSchedule
	//	for cur < params.EndTaskSchedule {
	//		cur
	//		hours = append(hours, cur)
	//	}

	return []Time{Date(2015, 02, 16, 0, 0, 0, 0, UTC)}
}
