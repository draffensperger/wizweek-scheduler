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
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	. "time"
)

// Monday Feb 9 - Saturday, Feb 21

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
		var p TaskParams
		err := ParseTaskParams(sampleJSON, &p)
		So(err, ShouldBeNil)

		Convey("Values are parsed correctly", func() {
			So(p.TimeZoneName, ShouldEqual, "America/New_York")
			EST, err = LoadLocation("America/New_York")
			So(err, ShouldBeNil)
			So(p.Location, ShouldResemble, EST)

			So(p.StartTaskSchedule, ShouldResemble, DateHour(2015, 2, 16, 9))
			So(p.EndTaskSchedule, ShouldResemble, DateHour(2015, 2, 20, 17))
			So(len(p.Tasks), ShouldEqual, 2)

			task0 := p.Tasks[0]
			So(task0.Title, ShouldEqual, "Newsletter")
			So(task0.EstimatedHours, ShouldEqual, 6)
			So(task0.Reward, ShouldEqual, 6)
			So(task0.Deadline, ShouldResemble, DateHour(2015, 2, 16, 17))

			task1 := p.Tasks[1]
			So(task1.Title, ShouldEqual, "Reimbursements")
			So(task1.EstimatedHours, ShouldEqual, 1)
			So(task1.Reward, ShouldEqual, 3)
			So(task1.Deadline, ShouldResemble, DateHour(2015, 2, 17, 17))

			So(len(p.WeeklyTaskBlocks), ShouldEqual, 7)
			for i, taskBlockLen := range []int{0, 1, 1, 1, 1, 1, 0} {
				So(len(p.WeeklyTaskBlocks[i]), ShouldEqual, taskBlockLen)
				if taskBlockLen > 0 {
					taskBlock := p.WeeklyTaskBlocks[i][0]
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
		[{"start": "13:00", "end": "15:00"}],
		[{"start": "18:00", "end": "19:00"}],
		[],
		[{"start": "9:00", "end": "10:00"}, {"start": "15:00", "end": "17:00"}],
		[]
	],	
	"appointments": [
		{ "title": "Meeting1", "start": "2015-02-16T00:00:00Z", "end": "2015-02-16T00:00:00Z" },
		{ "title": "Meeting2", "start": "2015-02-16T00:00:00Z", "end": "2015-02-16T00:00:00Z" }
	],
	"tasks": [	],
	"startTaskSchedule": "2015-02-16T00:00:00Z",
	"endTaskSchedule": "2015-02-20T00:00:00Z"
}`

func TestTaskHours(t *testing.T) {
	Convey("With task params specified it produces a series of task hour start times", t, func() {
		var params TaskParams
		ParseTaskParams(json2, &params)
		hours := params.TaskHours()
		So(len(hours), ShouldEqual, 1)
	})
}
