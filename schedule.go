package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/draffensperger/golp"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	. "time"
)

func main() {
	http.HandleFunc("/", computeScheduleHandler)

	listen := os.Getenv("PORT")
	if listen == "" {
		listen = ":8000"
	} else {
		listen = ":" + listen
	}
	fmt.Printf("Listening on %v\n", listen)
	log.Fatal(http.ListenAndServe(listen, nil))
}

func computeScheduleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Must use POST", http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	scheduleJSON, err := parseAndComputeSchedule(body)
	if err != nil {
		errJSON, jsonMarshalErr := json.Marshal(map[string]string{"err": err.Error()})
		if jsonMarshalErr != nil {
			http.Error(w, jsonMarshalErr.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(errJSON)
		return
	}

	_, err = w.Write(scheduleJSON)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func parseAndComputeSchedule(paramsJSON []byte) ([]byte, error) {
	var tp TaskParams
	if err := parseTaskParams(paramsJSON, &tp); err != nil {
		return nil, err
	}
	if err := tp.calcSchedule(); err != nil {
		return nil, err
	}
	return tp.taskScheduleJSON()
}

func parseTaskParams(paramsJSON []byte, tp *TaskParams) error {
	if err := json.Unmarshal(paramsJSON, tp); err != nil {
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

func (tp *TaskParams) calcSchedule() error {
	if err := tp.deadlineInPastErr(); err != nil {
		return err
	}

	if err := tp.setupLP(); err != nil {
		return err
	}

	tp.lp.SetVerboseLevel(golp.IMPORTANT)
	//tp.lp.WriteToStdout()
	ret := tp.lp.Solve()
	if ret != 0 {
		return errors.New(`Could not solve linear program`)
	}

	if err := tp.interpretTaskSchedule(); err != nil {
		return err
	}
	tp.formatTaskEvents()

	return nil
}

func (tp *TaskParams) taskScheduleJSON() ([]byte, error) {
	return json.MarshalIndent(tp.TaskEvents, "", "  ")
}

func (tp *TaskParams) localizeTimes() {
	tp.StartTaskSchedule = tp.StartTaskSchedule.In(tp.Location)
	tp.EndTaskSchedule = tp.EndTaskSchedule.In(tp.Location)

	for i := 0; i < len(tp.Tasks); i++ {
		tp.Tasks[i].Deadline = tp.Tasks[i].Deadline.In(tp.Location)
		tp.Tasks[i].StartOnOrAfter = tp.Tasks[i].StartOnOrAfter.In(tp.Location)
	}

	for i := 0; i < len(tp.Appointments); i++ {
		tp.Appointments[i].Start = tp.Appointments[i].Start.In(tp.Location)
		tp.Appointments[i].End = tp.Appointments[i].End.In(tp.Location)
	}
}

type TaskParams struct {
	TimeZoneName string `json:"timeZone"`
	*Location
	WeeklyTaskBlocks  [][]TimeBlock
	Tasks             []Task
	Appointments      []Appointment
	StartTaskSchedule Time
	EndTaskSchedule   Time
	TaskHours         []Time
	lp                *golp.LP
	TaskSchedule      []*Task
	TaskEvents        []TaskEvent
}

type Appointment struct {
	Title string
	Start Time
	End   Time
}

type TaskEvent struct {
	*Task  `json:"-"`
	Title  string `json:"title"`
	Start  Time   `json:"start"`
	End    Time   `json:"end"`
	Finish bool   `json:"finish"`
}

type Task struct {
	Title                   string
	EstimatedHours          float64
	Reward                  float64
	Deadline                Time
	DeadlineHourIndex       int
	StartOnOrAfter          Time
	StartOnOrAfterHourIndex int
	hoursScheduled          float64
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
			if !tp.appointmentInRange(t, hourAhead) {
				taskHours = append(taskHours, t)
			}
			t = hourAhead
		}
		hourAhead = t.Add(Hour)
	}

	tp.TaskHours = taskHours
}

// Could probably be made more efficient
func (tp *TaskParams) appointmentInRange(start, end Time) bool {
	for _, appt := range tp.Appointments {
		if appt.Start.Before(end) && appt.End.After(start) {
			return true
		}
	}
	return false
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
			return errors.New("Deadline in the past for task: " + task.Title)
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

func (tp *TaskParams) setupLP() error {
	ncol := len(tp.Tasks) * len(tp.TaskHours)

	tp.lp = golp.NewLP(0, ncol)
	tp.setColNames()
	tp.addHourConstraints()
	tp.addTaskConstraints()
	tp.addDeadlineConstraints()
	tp.addStartContraints()
	tp.addObjectiveFunction()

	return nil
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
		if task.DeadlineHourIndex < len(tp.TaskHours) && task.DeadlineHourIndex >= 0 {
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

func (tp *TaskParams) interpretTaskSchedule() error {
	vars := tp.lp.GetVariables()
	tp.TaskSchedule = make([]*Task, len(tp.TaskHours))
	for hour := 0; hour < len(tp.TaskHours); hour++ {
		for taskNum := 0; taskNum < len(tp.Tasks); taskNum++ {
			val := vars[tp.col(hour, taskNum)]

			delta := 0.001
			if math.Abs(val-1.0) < delta {
				tp.TaskSchedule[hour] = &tp.Tasks[taskNum]
			} else if math.Abs(val) > delta {
				return errors.New("Linear program assign time value that is not 1.0 or 0.0")
			}
		}
	}
	return nil
}

func (tp *TaskParams) formatTaskEvents() {
	tp.TaskEvents = make([]TaskEvent, 0)

	var hourAhead Time
	var prevTask *Task
	for i, task := range tp.TaskSchedule {
		if task != nil {
			if prevTask != task || tp.TaskHours[i].After(hourAhead) {
				var newEvent TaskEvent
				newEvent.Start = tp.TaskHours[i]
				newEvent.Task = task
				newEvent.Title = task.Title
				tp.TaskEvents = append(tp.TaskEvents, newEvent)
			}

			event := &tp.TaskEvents[len(tp.TaskEvents)-1]
			task.hoursScheduled++
			if task.hoursScheduled >= task.EstimatedHours {
				event.Finish = true
			}
			hourAhead = tp.TaskHours[i].Add(Hour)
			event.End = hourAhead
		}
		prevTask = task
	}
}
