package scheduler

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func test() {}

func TestIsRunning(t *testing.T) {

	fn := func() {
		time.Sleep(35 * time.Millisecond)
	}

	job, err := Every(1).Seconds().Run(fn)
	assert.Nil(t, err)
	assert.NotNil(t, job)

	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, true, job.IsRunning())
	job.SkipWait <- true
	assert.Equal(t, true, job.IsRunning())

	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, false, job.IsRunning())
}

func TestExecution(t *testing.T) {
	c := make(chan bool)
	fn := func() {
		c <- true
	}
	job, err := Every(1).Seconds().Run(fn)
	assert.Nil(t, err)
	assert.NotNil(t, job)
	select {
	case <-c:
	case <-time.After(2 * time.Second):
		t.Error("Didn't Execute")
	}
}

func TestDoNotWait(t *testing.T) {
	c := make(chan bool)
	fn := func() {
		c <- true
	}
	job, err := Every(2).Seconds().Run(fn)
	assert.Nil(t, err)
	assert.NotNil(t, job)
	job.SkipWait <- true
	select {
	case <-c:
	case <-time.After(1 * time.Second):
		t.Error("Didn't Execute")
	}
}

func TestCancelExecution(t *testing.T) {
	c := make(chan bool)
	fn := func() {
		c <- true
	}
	job, err := Every(1).Seconds().Run(fn)
	assert.Nil(t, err)
	assert.NotNil(t, job)
	job.Quit <- true
	select {
	case <-c:
		t.Error("Shouldn't have Execute")
	case <-time.After(2 * time.Second):
	}
}

func testDay(t *testing.T, job *Job, err error, date time.Time, hour, min, sec int) {
	assert.Nil(t, err)

	actual, err := job.schedule.nextRun()
	assert.Nil(t, err)
	runTime := time.Now().Add(actual)
	assert.Equal(t, date.Day(), runTime.Day())
	assert.Equal(t, date.Month(), runTime.Month())
	assert.Equal(t, date.Year(), runTime.Year())
	assert.Equal(t, hour, runTime.Hour())
	assert.Equal(t, min, runTime.Minute())
	assert.Equal(t, sec, runTime.Second())
}

func TestEveryDay(t *testing.T) {
	job, err := Every().Day().Run(test)
	testDay(t, job, err, time.Now().AddDate(0, 0, 1), 0, 0, 0)
}

func TestEveryAtAfter(t *testing.T) {
	tAdd := time.Now().Add(1 * time.Second)
	h := tAdd.Hour()
	m := tAdd.Minute()
	s := tAdd.Second()
	hourStr := fmt.Sprintf("%v:%v:%v", h, m, s)
	job, err := Every().Day().At(hourStr).Run(test)
	testDay(t, job, err, time.Now(), h, m, s)
}

func TestEveryAtBefore(t *testing.T) {
	tAdd := time.Now().Add(-1 * time.Second)
	h := tAdd.Hour()
	m := tAdd.Minute()
	s := tAdd.Second()
	hourStr := fmt.Sprintf("%v:%v:%v", h, m, s)
	job, err := Every().Day().At(hourStr).Run(test)
	testDay(t, job, err, time.Now().AddDate(0, 0, 1), h, m, s)
}

func TestEveryAtHour(t *testing.T) {
	hourStr := "08"
	job, err := Every().Day().At(hourStr).Run(test)
	assert.Nil(t, err)
	actual, err := job.schedule.nextRun()
	assert.Nil(t, err)
	runTime := time.Now().Add(actual)
	assert.Equal(t, 8, runTime.Hour())
	assert.Equal(t, 0, runTime.Minute())
	assert.Equal(t, 0, runTime.Second())
}

func TestEveryAtHourMin(t *testing.T) {
	hourStr := "08:30"
	job, err := Every().Day().At(hourStr).Run(test)
	assert.Nil(t, err)
	actual, err := job.schedule.nextRun()
	assert.Nil(t, err)
	runTime := time.Now().Add(actual)
	assert.Equal(t, 8, runTime.Hour())
	assert.Equal(t, 30, runTime.Minute())
	assert.Equal(t, 0, runTime.Second())
}

func testWeekday(t *testing.T, job *Job, weekday time.Weekday) {
	actual, err := job.schedule.nextRun()
	assert.Nil(t, err)
	runTime := time.Now().Add(actual)
	assert.Equal(t, weekday, runTime.Weekday())
	assert.Equal(t, 0, runTime.Hour())
	assert.Equal(t, 0, runTime.Minute())
	assert.Equal(t, 0, runTime.Second())

}

func TestEveryAtWeeklyAfter(t *testing.T) {
	tAdd := time.Now().Add(1 * time.Second)
	h := tAdd.Hour()
	m := tAdd.Minute()
	s := tAdd.Second()
	hourStr := fmt.Sprintf("%v:%v:%v", h, m, s)
	job, err := Every().Monday().At(hourStr).Run(test)
	assert.Nil(t, err)
	actual, err := job.schedule.nextRun()
	assert.Nil(t, err)
	runTime := time.Now().Add(actual)
	assert.Equal(t, time.Monday, runTime.Weekday())
	assert.Equal(t, h, runTime.Hour())
	assert.Equal(t, m, runTime.Minute())
	assert.Equal(t, s, runTime.Second())
}

func TestEveryAtWeeklyBefore(t *testing.T) {
	tAdd := time.Now().Add(-1 * time.Second)
	h := tAdd.Hour()
	m := tAdd.Minute()
	s := tAdd.Second()
	hourStr := fmt.Sprintf("%v:%v:%v", h, m, s)
	job, err := Every().Monday().At(hourStr).Run(test)
	assert.Nil(t, err)
	actual, err := job.schedule.nextRun()
	assert.Nil(t, err)
	runTime := time.Now().Add(actual)
	assert.Equal(t, time.Monday, runTime.Weekday())
	assert.Equal(t, h, runTime.Hour())
	assert.Equal(t, m, runTime.Minute())
	assert.Equal(t, s, runTime.Second())
}

func TestBadAtStr1(t *testing.T) {
	job, err := Every().Monday().At("0A:30:00").Run(test)
	assert.Nil(t, job)
	assert.NotNil(t, err)
}

func TestBadAtStr2(t *testing.T) {
	job, err := Every().Monday().At("00:3A:00").Run(test)
	assert.Nil(t, job)
	assert.NotNil(t, err)
}

func TestBadAtStr3(t *testing.T) {
	job, err := Every().Monday().At("00:30:0A").Run(test)
	assert.Nil(t, job)
	assert.NotNil(t, err)
}

func TestBadAtStr4(t *testing.T) {
	job, err := Every().Monday().At("25:30:00").Run(test)
	assert.Nil(t, job)
	assert.NotNil(t, err)
}

func TestBadChain1(t *testing.T) {
	job, err := Every(1).Day().At("1").Run(test)
	assert.Nil(t, job)
	assert.NotNil(t, err)
}

func TestBadChain2(t *testing.T) {
	job, err := Every(1).Day().Run(test)
	assert.Nil(t, job)
	assert.NotNil(t, err)
}

func TestBadChain3(t *testing.T) {
	job, err := Every(1).Seconds().At("08:00:00").Run(test)
	assert.Nil(t, job)
	assert.NotNil(t, err)
}

func TestBadChain4(t *testing.T) {
	job, err := Every(1).Seconds().Monday().Run(test)
	assert.Nil(t, job)
	assert.NotNil(t, err)
}

func TestBadChain5(t *testing.T) {
	job, err := Every().Monday().NotImmediately().Run(test)
	assert.Nil(t, job)
	assert.NotNil(t, err)
}

func TestBadEvery(t *testing.T) {
	job, err := Every(1, 2).Seconds().Run(test)
	assert.Nil(t, job)
	assert.NotNil(t, err)
}

func TestEveryMonday(t *testing.T) {
	job, err := Every().Monday().Run(test)
	assert.Nil(t, err)
	testWeekday(t, job, time.Monday)
}

func TestEveryTuesday(t *testing.T) {
	job, err := Every().Tuesday().Run(test)
	assert.Nil(t, err)
	testWeekday(t, job, time.Tuesday)
}

func TestEveryWednesday(t *testing.T) {
	job, err := Every().Wednesday().Run(test)
	assert.Nil(t, err)
	testWeekday(t, job, time.Wednesday)
}

func TestEveryThursday(t *testing.T) {
	job, err := Every().Thursday().Run(test)
	assert.Nil(t, err)
	testWeekday(t, job, time.Thursday)
}

func TestEveryFriday(t *testing.T) {
	job, err := Every().Friday().Run(test)
	assert.Nil(t, err)
	testWeekday(t, job, time.Friday)
}

func TestEverySaturday(t *testing.T) {
	job, err := Every().Saturday().Run(test)
	assert.Nil(t, err)
	testWeekday(t, job, time.Saturday)
}

func TestEverySunday(t *testing.T) {
	job, err := Every().Sunday().Run(test)
	assert.Nil(t, err)
	testWeekday(t, job, time.Sunday)
}

func testEveryX(t *testing.T, job *Job, expected time.Duration, immediate bool) {
	actual, err := job.schedule.nextRun()
	assert.Nil(t, err)
	if immediate {
		assert.Equal(t, time.Duration(0), actual)
	}
	actual, err = job.schedule.nextRun()
	assert.Nil(t, err)
	assert.Equal(t, expected, actual)
}

func TestEverySeconds(t *testing.T) {
	var units int
	units = 3
	expected := 3 * time.Second
	job := Every(units).Seconds()
	testEveryX(t, job, expected, true)
}

func TestEveryMinutes(t *testing.T) {
	var units int
	units = 3
	expected := 3 * time.Minute
	job := Every(units).Minutes()
	testEveryX(t, job, expected, true)
}

func TestEveryHours(t *testing.T) {
	var units int
	units = 3
	expected := 3 * time.Hour
	job := Every(units).Hours()
	testEveryX(t, job, expected, true)
}

func TestEveryHoursNotImmediately(t *testing.T) {
	var units int
	units = 3
	expected := 3 * time.Hour
	job := Every(units).Hours().NotImmediately()
	testEveryX(t, job, expected, false)
}

func TestBadRecurrent(t *testing.T) {
	var units int
	units = 0
	job, err := Every(units).Seconds().Run(test)
	assert.Nil(t, job)
	assert.NotNil(t, err)
}
