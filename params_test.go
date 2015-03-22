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
	Convey("When task params parsed from JSON", t, func() {
		var p TaskParams
		err := ParseTaskParams(sampleJSON, &p)
		So(err, ShouldBeNil)

		Convey("Values are parsed correctly", func() {
			So(p.StartTaskSchedule, ShouldResemble, Date(2015, 02, 16, 0, 0, 0, 0, UTC))
			So(p.EndTaskSchedule, ShouldResemble, Date(2015, 02, 20, 0, 0, 0, 0, UTC))
			So(len(p.Tasks), ShouldEqual, 2)

			task0 := p.Tasks[0]
			So(task0.Title, ShouldEqual, "Newsletter")
			So(task0.EstimatedHours, ShouldEqual, 6)
			So(task0.Reward, ShouldEqual, 6)
			So(task0.Deadline, ShouldResemble, Date(2015, 02, 16, 21, 0, 0, 0, UTC))

			task1 := p.Tasks[1]
			So(task1.Title, ShouldEqual, "Reimbursements")
			So(task1.EstimatedHours, ShouldEqual, 1)
			So(task1.Reward, ShouldEqual, 3)
			So(task1.Deadline, ShouldResemble, Date(2015, 02, 17, 21, 0, 0, 0, UTC))

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

//func TestTaskHours(t *testing.T) {
//	var params TaskParams
//	ParseTaskParams(sampleJSON, &params)
//	hours := params.TaskHours
//	assert.Equal()
//}
