/*

current:
https://github.com/onsi/ginkgo
https://github.com/onsi/gomega
https://github.com/smartystreets/goconvey
https://github.com/stretchr/testify/
https://github.com/franela/goblin

maybe:
https://github.com/pranavraja/zen
https://github.com/go-check/check
https://github.com/azer/mao

old:

 https://github.com/remogatto/prettytest
 https://github.com/bmatsuo/go-spec
 https://github.com/orfjackal/gospec

meta:

https://github.com/shageman/gotestit
*/

package schedule

// . "https://github.com/stretchr/testify/assert"
import (
	"github.com/k0kubun/pp"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	. "time"
)

func p(x interface{}) {
	pp.Println(x)
}

func pf(f string, x interface{}) {
	pp.Printf(f+" %v\n", x)
}

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
	"appointments": [	],
	"tasks": [
		{"title": "Newsletter", "estimatedHours": 6, "reward": 6, "deadline": "2015-02-16T22:00:00Z"},
		{"title": "Reimbursements", "estimatedHours": 1, "reward": 3, "deadline": "2015-02-17T22:00:00Z"}
	],
	"startTaskSchedule": "2015-02-16T14:00:00Z",
	"endTaskSchedule": "2015-02-20T22:00:00Z"
}`

var EST *Location

func DateHour(year int, month Month, day, hour int) Time {
	return Date(year, month, day, hour, 0, 0, 0, EST)
}

func TestTaskParams(t *testing.T) {
	Convey("When task params parsed from JSON", t, func() {
		var tp TaskParams
		err := ParseTaskParams(sampleJSON, &tp)
		So(err, ShouldBeNil)

		Convey("Values are parsed correctly", func() {
			So(tp.TimeZoneName, ShouldEqual, "America/New_York")
			EST, err = LoadLocation("America/New_York")
			So(err, ShouldBeNil)
			So(tp.Location, ShouldResemble, EST)

			So(tp.StartTaskSchedule, ShouldResemble, DateHour(2015, 2, 16, 9))
			So(tp.EndTaskSchedule, ShouldResemble, DateHour(2015, 2, 20, 17))
			So(len(tp.Tasks), ShouldEqual, 2)

			task0 := tp.Tasks[0]
			So(task0.Title, ShouldEqual, "Newsletter")
			So(task0.EstimatedHours, ShouldEqual, 6)
			So(task0.Reward, ShouldEqual, 6)
			So(task0.Deadline, ShouldResemble, DateHour(2015, 2, 16, 17))

			task1 := tp.Tasks[1]
			So(task1.Title, ShouldEqual, "Reimbursements")
			So(task1.EstimatedHours, ShouldEqual, 1)
			So(task1.Reward, ShouldEqual, 3)
			So(task1.Deadline, ShouldResemble, DateHour(2015, 2, 17, 17))

			So(len(tp.WeeklyTaskBlocks), ShouldEqual, 7)
			for i, taskBlockLen := range []int{0, 1, 1, 1, 1, 1, 0} {
				So(len(tp.WeeklyTaskBlocks[i]), ShouldEqual, taskBlockLen)
				if taskBlockLen > 0 {
					taskBlock := tp.WeeklyTaskBlocks[i][0]
					So(taskBlock.Start.Hour(), ShouldEqual, 10)
					So(taskBlock.End.Hour(), ShouldEqual, 16)
				}
			}
		})
	})
}

const json2 = `{	
	"timeZone": "America/New_York",
	"weeklyTaskBlocks": [
		[],
		[{"start": "10:00", "end": "12:00"}],
		[{"start": "9:00", "end": "10:00"}, {"start": "11:30", "end": "14:30"}],
		[],
		[],
		[{"start": "16:00", "end": "18:00"}],
		[]
	],	
	"appointments": [	],
	"tasks": [
		{"title": "Newsletter", "estimatedHours": 6, "reward": 6, "deadline": "2015-02-16T22:00:00Z"},
		{"title": "Reimbursements", "estimatedHours": 1, "reward": 3, "deadline": "2015-02-17T22:00:00Z"}
	],
	"startTaskSchedule": "2015-02-16T14:00:00Z",
	"endTaskSchedule": "2015-02-25T22:00:00Z"
}`

func TestTaskHours(t *testing.T) {
	Convey("With task params specified it produces a series of task hour start times", t, func() {
		var tp TaskParams
		ParseTaskParams(json2, &tp)
		hours := tp.TaskHours
		So(len(hours), ShouldEqual, 14)

		EST, err := LoadLocation("America/New_York")
		So(err, ShouldBeNil)
		So(hours, ShouldResemble, []Time{
			Date(2015, 2, 16, 10, 0, 0, 0, EST),
			Date(2015, 2, 16, 11, 0, 0, 0, EST),
			Date(2015, 2, 17, 9, 0, 0, 0, EST),
			Date(2015, 2, 17, 11, 30, 0, 0, EST),
			Date(2015, 2, 17, 12, 30, 0, 0, EST),
			Date(2015, 2, 17, 13, 30, 0, 0, EST),
			Date(2015, 2, 20, 16, 0, 0, 0, EST),
			Date(2015, 2, 20, 17, 0, 0, 0, EST),
			Date(2015, 2, 23, 10, 0, 0, 0, EST),
			Date(2015, 2, 23, 11, 0, 0, 0, EST),
			Date(2015, 2, 24, 9, 0, 0, 0, EST),
			Date(2015, 2, 24, 11, 30, 0, 0, EST),
			Date(2015, 2, 24, 12, 30, 0, 0, EST),
			Date(2015, 2, 24, 13, 30, 0, 0, EST),
		})
	})
}

const json3 = `{	
	"timeZone": "America/New_York",
	"weeklyTaskBlocks": [
		[],
		[{"start": "10:00", "end": "12:00"}],
		[{"start": "9:00", "end": "10:00"}, {"start": "11:30", "end": "14:30"}],
		[],
		[],
		[{"start": "16:00", "end": "18:00"}],
		[]
	],	
	"appointments": [	],
	"tasks": [
		{"title": "Newsletter", "estimatedHours": 2, "reward": 6, "deadline": "2015-02-20T22:00:00Z", "startOnOrAfter": "2015-02-17T15:00:00Z"},
		{"title": "Reimbursements", "estimatedHours": 1, "reward": 3, "deadline": "2015-02-23T22:00:00Z"},
		{"title": "Plan study", "estimatedHours": 1, "reward": 3, "startOnOrAfter": "2015-02-18T15:00:00Z"},
		{"title": "Past due", "estimatedHours": 1, "reward": 3, "deadline": "2015-01-01T22:00:00Z"},
		{"title": "Admin work", "estimatedHours": 1, "reward": 3}				
	],
	"startTaskSchedule": "2015-02-16T14:00:00Z",
	"endTaskSchedule": "2015-02-25T22:00:00Z"
}`

func TestDeadlineAndOnOrAfter(t *testing.T) {
	Convey("With task params specified it sets the hour indicies for deadlines and start on or after", t, func() {
		var tp TaskParams
		ParseTaskParams(json3, &tp)
		tasks := tp.Tasks

		So(len(tasks), ShouldEqual, 5)
		So(tasks[0].DeadlineHourIndex, ShouldEqual, 6)
		So(tasks[0].StartOnOrAfterHourIndex, ShouldEqual, 3)
		So(tasks[1].DeadlineHourIndex, ShouldEqual, 9)
		So(tasks[1].StartOnOrAfterHourIndex, ShouldEqual, 0)
		So(tasks[2].DeadlineHourIndex, ShouldEqual, len(tp.TaskHours))
		So(tasks[2].StartOnOrAfterHourIndex, ShouldEqual, 6)
		So(tasks[3].DeadlineHourIndex, ShouldEqual, -1)
		So(tasks[3].StartOnOrAfterHourIndex, ShouldEqual, 0)
		So(tasks[4].DeadlineHourIndex, ShouldEqual, len(tp.TaskHours))
		So(tasks[4].StartOnOrAfterHourIndex, ShouldEqual, 0)
	})
}
