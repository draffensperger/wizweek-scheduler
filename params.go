package schedule

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/draffensperger/golp"
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

	tp.setupLP()

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
	lp                *golp.LP
	TaskSchedule      []*Task
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

func (tp TaskParams) deadlineInPastErr() error {
	for _, task := range tp.Tasks {
		if task.DeadlineHourIndex < 0 {
			return errors.New(`Deadline in the past for task`)
		}
	}
	return nil
}

func (tp *TaskParams) calculateSchedule() error {
	if err := tp.deadlineInPastErr(); err != nil {
		return err
	}
	return nil
}

func (tp *TaskParams) setupLP() {
	ncol := len(tp.Tasks) * len(tp.TaskHours)

	tp.lp = golp.NewLP(0, ncol)
	tp.setColNames()
	tp.addHourConstraints()
	tp.addTaskConstraints()
	tp.addDeadlineConstraints()
	tp.addStartContraints()
	tp.addObjectiveFunction()

	fmt.Println("\n")
	// fmt.Println("LP formulation:")
	// tp.lp.WriteToStdout()

	ret := tp.lp.Solve()
	fmt.Printf("Solve returned: %v\n", ret)

	//obj := tp.lp.GetObjective()
	//fmt.Printf("Objective value: %v\n", obj)

}

func (tp *TaskParams) getTaskSchedule() {
	vars := tp.lp.GetVariables()
	tp.TaskSchedule = make([]*Task, len(tp.TaskHours))
	for hour := 0; hour < len(tp.TaskHours); hour++ {
		for taskNum := 0; taskNum < len(tp.Tasks); taskNum++ {
			val := vars[tp.col(hour, taskNum)]
			if val == 1.0 {
				tp.TaskSchedule[hour] = &tp.Tasks[taskNum]
			}
			// nameStr := "h" + strconv.Itoa(hour) + "_t" + strconv.Itoa(taskNum)
			// fmt.Printf("%v: %v\n", nameStr, val)
		}
	}
}

func (tp *TaskParams) writeLPOutput() {

}

func (tp TaskParams) col(hour, taskNum int) int {
	return hour*len(tp.Tasks) + taskNum
}

func (tp *TaskParams) setColNames() {
	for hour := 0; hour < len(tp.TaskHours); hour++ {
		for taskNum := 0; taskNum < len(tp.Tasks); taskNum++ {
			tp.lp.SetColName(tp.col(hour, taskNum), "h"+strconv.Itoa(hour)+"_t"+strconv.Itoa(taskNum))
		}
	}
}

func (tp *TaskParams) addHourConstraints() {
	// Total tasks done in a hour must be <= 1
	for hour := 0; hour < len(tp.TaskHours); hour++ {
		entries := make([]golp.Entry, len(tp.Tasks))
		for taskNum := 0; taskNum < len(tp.Tasks); taskNum++ {
			entries[taskNum].Col = tp.col(hour, taskNum)
			entries[taskNum].Val = 1.0
		}
		tp.lp.AddConstraintSparse(entries, golp.LE, 1.0)
	}
}

func (tp *TaskParams) addTaskConstraints() {
	// Total amount done on each task must be <= task.EstimatedHours
	for taskNum, task := range tp.Tasks {
		entries := make([]golp.Entry, len(tp.TaskHours))
		for hour := 0; hour < len(tp.TaskHours); hour++ {
			entries[hour].Col = tp.col(hour, taskNum)
			entries[hour].Val = 1.0
		}
		tp.lp.AddConstraintSparse(entries, golp.LE, task.EstimatedHours)
	}
}

func (tp *TaskParams) addDeadlineConstraints() {
	// Total amount done on task with deadline up to the deadline hour index must equal the estimated hours
	for taskNum, task := range tp.Tasks {
		if task.DeadlineHourIndex < len(tp.Tasks) && task.DeadlineHourIndex >= 0 {
			entries := make([]golp.Entry, task.DeadlineHourIndex+1)
			for hour := 0; hour <= task.DeadlineHourIndex; hour++ {
				entries[hour].Col = tp.col(hour, taskNum)
				entries[hour].Val = 1.0
			}
			tp.lp.AddConstraintSparse(entries, golp.EQ, task.EstimatedHours)
		}
	}
}

func (tp *TaskParams) addStartContraints() {
	// Total amount done on task with deadline up to the deadline hour index must equal zero
	for taskNum, task := range tp.Tasks {
		if task.StartOnOrAfterHourIndex > 0 {
			entries := make([]golp.Entry, task.StartOnOrAfterHourIndex)
			for hour := 0; hour < task.StartOnOrAfterHourIndex; hour++ {
				entries[hour].Col = tp.col(hour, taskNum)
				entries[hour].Val = 1.0
			}
			tp.lp.AddConstraintSparse(entries, golp.EQ, 0.0)
		}
	}
}

func (tp *TaskParams) addObjectiveFunction() {
	// Objective function
	decayRate := 0.95
	curHourValue := 1.0
	row := make([]float64, len(tp.Tasks)*len(tp.TaskHours))
	for hour := 0; hour < len(tp.TaskHours); hour++ {
		for taskNum, task := range tp.Tasks {
			row[tp.col(hour, taskNum)] = curHourValue * task.Reward / task.EstimatedHours
		}
		curHourValue *= decayRate
	}
	tp.lp.SetObjFn(row, true)
}
