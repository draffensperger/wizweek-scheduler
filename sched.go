/*
Next steps for the go project:

Create a git repository for it
Encapsulate LP and add a test case for it
Encapsulate the JSON processing and add a test case for it


Set up a Travis CI build that automatically deploys it to Google App Engine

*/

// You can edit this code!
// Click here and start typing.
package main

// typedef int (*intFunc) ();
//
// int
// bridge_int_func(intFunc f)
// {
//		return f();
// }
//
// int fortytwo()
// {
//	    return 42;
// }
//
//
//
/*
#cgo CFLAGS: -I./lp_solve
#cgo LDFLAGS: -L./lp_solve -llpsolve55 -Wl,-rpath=./lp_solve
#include <stdlib.h>
#include "lp_lib.h"
*/
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
	"unsafe"
)

/*
Go REPLs
http://play.golang.org/
https://github.com/vito/go-repl
https://github.com/sbinet/igo
https://github.com/sriram-srinivasan/gore
http://labix.org/hsandbox

*/

const SampleJSON = `{	
	"timeZone": "America/New_York",
	"weeklyTaskBlocks": [
		[],
		[{"start": "10:00", "end": "16:00"}],
		[{"start": "10:00", "end": "16:00"}],
		[{"start": "10:00", "end": "16:00"}],
		[{"start": "10:00", "end": "16:00"}],
		[{"start": "10:00", "end": "16:00"}],
		[]
	],	
	"appointments": [],
	"tasks": [
		{"title": "Newsletter", "estimatedHours": 6, "reward": 6, "deadline": "2015-02-16T21:00:00Z"},
		{"title": "Reimbursements", "estimatedHours": 1, "reward": 3, "deadline": "2015-02-17T21:00:00Z"},
		{"title": "FB Strategy Writeup", "estimatedHours": 1, "reward": 2}
	],
	"startTaskSchedule": "2015-02-16T00:00:00Z",
	"endTaskSchedule": "2015-02-20T00:00:00Z"
}`

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

type TimeBlock struct {
	Start TimeWithoutDate
	End   TimeWithoutDate
}

type Task struct {
	Title          string
	EstimatedHours float32
	Reward         float32
	Deadline       time.Time
	StartOnOrAfter time.Time
}

type TaskParams struct {
	TimeZone          string
	WeeklyTaskBlocks  [][]TimeBlock
	Tasks             []Task
	StartTaskSchedule time.Time
	EndTaskSchedule   time.Time
}

func main() {
	var err error

	var f interface{}
	err = json.Unmarshal([]byte(SampleJSON), &f)
	if err != nil {
		fmt.Printf(" %v", err)
	}

	var taskParams TaskParams
	err = json.Unmarshal([]byte(SampleJSON), &taskParams)
	if err != nil {
		fmt.Printf(" %v", err)
	}
	fmt.Println("Task params:")
	fmt.Printf("%+v", taskParams)

	formatLP(taskParams)
}

func formatLP(params TaskParams) {
	/*
		LP format:

		step 1: break tasks up into hourly chunks, (later on re-weight the longer tasks as less rewarding)


		variables are
			date_hour_taskpart = 1.0 means do taskpart at date and hour
			all of those date_hour_taskpart variables must be between 0 and 1
			sum of all date_hour_taskpart = 1.0 (can only do one hour of total work per hour)

		deadlines:
			if a task part has a deadline, then the sum of all its work times before that deadline
			must be 1.0

		on or after:
			if a task part has an on or after specified, the sum of work times before on or after must be 0

		reward for each hour:
			hour_reward = hour decay const * date_hour_taskpart * value/hr for task

	*/

	tasks := params.Tasks
	//horizonHours := 22 * 8
	horizonHours := 8

	ncol := C.int(len(tasks) * horizonHours)
	lp := C.make_lp(0, ncol)

	for hour := 0; hour < horizonHours; hour++ {
		for taskNum := 0; taskNum < len(tasks); taskNum++ {
			nameStr := "h" + strconv.Itoa(hour) + "_t" + strconv.Itoa(taskNum)
			name := C.CString(nameStr)
			colNum := hour*len(tasks) + taskNum + 1
			C.set_col_name(lp, C.int(colNum), name)
			C.free(unsafe.Pointer(name))
		}
	}

	C.set_add_rowmode(lp, 1)

	// Each variable must be between 0 and 1

	// Total tasks done in a hour must be <= 1
	for hour := 0; hour < horizonHours; hour++ {
		row := make([]C.double, len(tasks))
		colNums := make([]C.int, len(tasks))
		for taskNum := 0; taskNum < len(tasks); taskNum++ {
			colNums[taskNum] = C.int(len(tasks)*hour + taskNum + 1)
			row[taskNum] = 1.0
		}
		C.add_constraintex(lp, C.int(len(tasks)), &row[0], &colNums[0], C.LE, 1.0)
	}

	// Total amount done on each task must be <= task.EstimatedHours
	for taskNum, task := range tasks {
		row := make([]C.double, horizonHours)
		colNums := make([]C.int, horizonHours)
		for hour := 0; hour < horizonHours; hour++ {
			colNums[hour] = C.int(len(tasks)*hour + taskNum + 1)
			row[hour] = 1.0
		}
		C.add_constraintex(lp, C.int(horizonHours), &row[0], &colNums[0], C.LE, C.double(task.EstimatedHours))
	}
	C.set_add_rowmode(lp, 0)

	// Objective function
	decayRate := float32(0.95)
	curHourValue := float32(1.0)
	row := make([]C.double, len(tasks)*horizonHours+1)
	for hour := 0; hour < horizonHours; hour++ {
		for taskNum, task := range tasks {
			row[len(tasks)*hour+taskNum+1] = C.double(curHourValue * task.Reward / task.EstimatedHours)
		}
		curHourValue *= decayRate
	}

	C.set_obj_fn(lp, &row[0])

	C.set_maxim(lp)

	fmt.Println("\n")
	fmt.Println("LP formulation:")
	C.write_LP(lp, C.stdout)

	ret := C.solve(lp)
	fmt.Printf("Solve returned: %v\n", ret)

	obj := C.get_objective(lp)
	fmt.Printf("Objective value: %v\n", obj)

	C.get_variables(lp, &row[0])
	for hour := 0; hour < horizonHours; hour++ {
		for taskNum := 0; taskNum < len(tasks); taskNum++ {
			nameStr := "h" + strconv.Itoa(hour) + "_t" + strconv.Itoa(taskNum)
			val := row[len(tasks)*hour+taskNum]
			fmt.Printf("%v: %v\n", nameStr, val)
		}
	}

	C.delete_lp(lp)
}

func solve() {
	fmt.Println("Hello, 世界")
	Seed(44)
	fmt.Println(Random())
	fmt.Println(Random())
	fmt.Println(Random())

	var ncol C.int
	ncol = 2
	lp := C.make_lp(0, ncol)
	C.set_col_name(lp, 1, C.CString("x"))
	C.set_col_name(lp, 2, C.CString("y"))

	C.set_add_rowmode(lp, 1)

	colno := []C.int{1, 2}

	var j C.int
	j = 2

	row := []C.double{120.0, 210.0}
	C.add_constraintex(lp, j, &row[0], &colno[0], C.LE, 15000)

	row = []C.double{110.0, 30.0}
	C.add_constraintex(lp, j, &row[0], &colno[0], C.LE, 4000)

	row = []C.double{1.0, 1.0}
	C.add_constraintex(lp, j, &row[0], &colno[0], C.LE, 75)

	C.set_add_rowmode(lp, 0)

	row = []C.double{143.0, 60.0}
	C.set_obj_fnex(lp, j, &row[0], &colno[0])
	C.set_maxim(lp)
	C.write_LP(lp, C.stdout)
	C.set_verbose(lp, C.IMPORTANT)

	ret := C.solve(lp)
	fmt.Println("Solve returned: ")
	fmt.Println(ret)

	obj := C.get_objective(lp)
	fmt.Println("Objective value: ")
	fmt.Println(obj)

	C.get_variables(lp, &row[0])
	fmt.Println("x: ")
	fmt.Println(row[0])

	fmt.Println("y: ")
	fmt.Println(row[1])

	C.delete_lp(lp)

	f := C.intFunc(C.fortytwo)
	fmt.Println(int(C.bridge_int_func(f)))
}

func Random() int {
	return int(C.random())
}

func Seed(i int) {
	C.srandom(C.uint(i))
}
