package main

import (
	"fmt"
	"github.com/draffensperger/golp"
	"strconv"
)

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

func main() {
	var err error

	var taskParams TaskParams
	err = ParseTaskParams(SampleJSON, &taskParams)
	if err != nil {
		fmt.Printf(" %v", err)
	}
	fmt.Println("Task params:")
	fmt.Printf("%+v\n", taskParams)

	formatLP(taskParams)
	fmt.Println("")
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

	ncol := len(tasks) * horizonHours

	lp := golp.NewLP(0, ncol)
	//lp.SetVerboseLevel(6)

	for hour := 0; hour < horizonHours; hour++ {
		for taskNum := 0; taskNum < len(tasks); taskNum++ {
			colNum := hour*len(tasks) + taskNum
			lp.SetColName(colNum, "h"+strconv.Itoa(hour)+"_t"+strconv.Itoa(taskNum))
		}
	}

	// Total tasks done in a hour must be <= 1
	for hour := 0; hour < horizonHours; hour++ {
		entries := make([]golp.Entry, len(tasks))
		for taskNum := 0; taskNum < len(tasks); taskNum++ {
			entries[taskNum].Col = len(tasks)*hour + taskNum
			entries[taskNum].Val = 1.0
		}
		lp.AddConstraintSparse(entries, golp.LE, 1.0)
	}

	// Total amount done on each task must be <= task.EstimatedHours
	for taskNum, task := range tasks {
		entries := make([]golp.Entry, horizonHours)
		for hour := 0; hour < horizonHours; hour++ {
			entries[hour].Col = len(tasks)*hour + taskNum
			entries[hour].Val = 1.0
		}
		lp.AddConstraintSparse(entries, golp.LE, task.EstimatedHours)
	}

	// Objective function
	decayRate := 0.95
	curHourValue := 1.0
	row := make([]float64, len(tasks)*horizonHours)
	for hour := 0; hour < horizonHours; hour++ {
		for taskNum, task := range tasks {
			row[len(tasks)*hour+taskNum] = curHourValue * task.Reward / task.EstimatedHours
		}
		curHourValue *= decayRate
	}

	lp.SetObjFn(row, true)

	fmt.Println("\n")
	fmt.Println("LP formulation:")
	lp.WriteToStdout()

	ret := lp.Solve()
	fmt.Printf("Solve returned: %v\n", ret)

	obj := lp.GetObjective()
	fmt.Printf("Objective value: %v\n", obj)

	vars := lp.GetVariables()
	for hour := 0; hour < horizonHours; hour++ {
		for taskNum := 0; taskNum < len(tasks); taskNum++ {
			nameStr := "h" + strconv.Itoa(hour) + "_t" + strconv.Itoa(taskNum)
			val := vars[len(tasks)*hour+taskNum]
			fmt.Printf("%v: %v\n", nameStr, val)
		}
	}
}
