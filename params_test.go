package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const sampleJSON = `{	
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
		{"title": "Reimbursements", "estimatedHours": 1, "reward": 3, "deadline": "2015-02-17T21:00:00Z"}
	],
	"startTaskSchedule": "2015-02-16T00:00:00Z",
	"endTaskSchedule": "2015-02-20T00:00:00Z"
}`

func TestTaskParams(t *testing.T) {
	var params TaskParams
	err := ParseTaskParams(sampleJSON, &params)
	assert.Nil(t, err)

	assert.Equal(t, time.Date(2015, 02, 16, 0, 0, 0, 0, time.UTC), params.StartTaskSchedule)
	assert.Equal(t, time.Date(2015, 02, 20, 0, 0, 0, 0, time.UTC), params.EndTaskSchedule)
	assert.Equal(t, 2, len(params.Tasks))

	task0 := params.Tasks[0]
	assert.Equal(t, "Newsletter", task0.Title)
	assert.Equal(t, 6, task0.EstimatedHours)
	assert.Equal(t, 6, task0.Reward)
	assert.Equal(t, time.Date(2015, 02, 16, 21, 0, 0, 0, time.UTC), task0.Deadline)

	task1 := params.Tasks[1]
	assert.Equal(t, "Reimbursements", task1.Title)
	assert.Equal(t, 1, task1.EstimatedHours)
	assert.Equal(t, 3, task1.Reward)
	assert.Equal(t, time.Date(2015, 02, 17, 21, 0, 0, 0, time.UTC), task1.Deadline)

	assert.Equal(t, 7, len(params.WeeklyTaskBlocks))
	for i, taskBlockLen := range []int{0, 1, 1, 1, 1, 1, 0} {
		assert.Equal(t, taskBlockLen, len(params.WeeklyTaskBlocks[i]))
		if taskBlockLen > 0 {
			taskBlock := params.WeeklyTaskBlocks[i][0]
			assert.Equal(t, 10, taskBlock.Start.Hour())
			assert.Equal(t, 16, taskBlock.End.Hour())
		}
	}
}
