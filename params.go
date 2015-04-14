package schedule

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	. "time"
)

func ParseTaskParams(paramsJSON string, tp *TaskParams) error {
	if err := json.Unmarshal([]byte(paramsJSON), tp); err != nil {
		return err
	}

	loc, err := LoadLocation(tp.TimeZoneName)
	if err != nil {
		return err
	}
	tp.Location = loc
	tp.localizeTimes()
	tp.calculateTaskHours()

	for i := 0; i < len(tp.Tasks); i++ {
		tp.Tasks[i].DeadlineHourIndex = tp.deadlineAsTaskHour(tp.Tasks[i].Deadline)
		tp.Tasks[i].StartOnOrAfterHourIndex = tp.onOrAfterAsTaskHour(tp.Tasks[i].StartOnOrAfter)
	}

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
	TaskHours         []Time
}

type Task struct {
	Title                   string
	EstimatedHours          float64
	Reward                  float64
	Deadline                Time
	DeadlineHourIndex       int
	StartOnOrAfter          Time
	StartOnOrAfterHourIndex int
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
	s = s[1 : len(s)-1] // Get rid of the enclosing quotes
	timeParts := strings.Split(s, ":")
	if len(timeParts) != 2 {
		return TimeParseError
	}
	hour, err := strconv.Atoi(timeParts[0])
	if err != nil {
		return err
	}
	minute, err := strconv.Atoi(timeParts[1])
	if err != nil {
		return err
	}

	t.Time = Date(0, 0, 0, hour, minute, 0, 0, UTC)
	return nil
}

func (tp TaskParams) moveTimeToNextBlock(t *Time) (blockEnd Time) {
	blockStart := Time{}
	blockEnd = Time{}
	for weekdays := 0; weekdays < 7; weekdays++ {
		for _, block := range tp.WeeklyTaskBlocks[t.Weekday()] {
			year, month, day := t.Truncate(Hour).Date()
			blockStart = Date(year, month, day, block.Start.Hour(), block.Start.Minute(), 0, 0, tp.Location)
			blockEnd = Date(year, month, day, block.End.Hour(), block.End.Minute(), 0, 0, tp.Location)

			if t.Before(blockStart) {
				*t = blockStart
			}
			if !t.Add(Hour).After(blockEnd) {
				return blockEnd
			}
		}
		*t = t.Add(Duration(24-t.Hour()) * Hour)
	}
	return blockEnd
}

func (tp *TaskParams) calculateTaskHours() {
	taskHours := make([]Time, 0)
	t := tp.StartTaskSchedule
	blockEnd := tp.moveTimeToNextBlock(&t)
	hourAhead := t.Add(Hour)

	for !hourAhead.After(tp.EndTaskSchedule) {
		if hourAhead.After(blockEnd) {
			blockEnd = tp.moveTimeToNextBlock(&t)
		} else {
			taskHours = append(taskHours, t)
			t = hourAhead
		}
		hourAhead = t.Add(Hour)
	}

	tp.TaskHours = taskHours
}

// Return the index for the start of the last hour that you could work on a task to finish it by time t
// Could be optimized to use binary search
func (tp TaskParams) deadlineAsTaskHour(deadline Time) int {
	if deadline.IsZero() {
		// If deadline is unspecified return the value just after the end
		return len(tp.TaskHours)
	}

	for taskHour := len(tp.TaskHours) - 1; taskHour >= 0; taskHour-- {
		hourAhead := tp.TaskHours[taskHour].Add(Hour)
		if !hourAhead.After(deadline) {
			return taskHour
		}
	}

	return -1 // Deadline is before the first task
}

// Return the index for the start of the first hour that you could work on a task to if you are only allowed to work on it starting on or after time t
// Could be optimized to use binary search
func (tp TaskParams) onOrAfterAsTaskHour(onOrAfter Time) int {
	if onOrAfter.IsZero() {
		return 0
	}
	for taskHour, taskHourTime := range tp.TaskHours {
		if !taskHourTime.Before(onOrAfter) {
			return taskHour
		}
	}
	return -1 // Can't start this task in the time horizon given
}
