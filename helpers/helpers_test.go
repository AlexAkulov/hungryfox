package helpers

import (
	"testing"
	"time"

	"github.com/AlexAkulov/hungryfox"
	. "github.com/smartystreets/goconvey/convey"
)

func TestParseDuration(t *testing.T) {
	Convey("1s", t, func() {
		result, err := ParseDuration("1s")
		So(err, ShouldBeNil)
		So(result, ShouldEqual, time.Duration(time.Second))
	})
	Convey("22m", t, func() {
		result, err := ParseDuration("22m")
		So(err, ShouldBeNil)
		So(result, ShouldEqual, time.Duration(time.Minute*22))
	})
	Convey("333h", t, func() {
		result, err := ParseDuration("333h")
		So(err, ShouldBeNil)
		So(result, ShouldEqual, time.Duration(time.Hour*333))
	})
	Convey("4444d", t, func() {
		result, err := ParseDuration("4444d")
		So(err, ShouldBeNil)
		So(result, ShouldEqual, time.Duration(time.Hour*24*4444))
	})
	Convey("5y", t, func() {
		result, err := ParseDuration("5y")
		So(err, ShouldBeNil)
		So(result, ShouldEqual, time.Duration(time.Hour*24*365*5))
	})
	Convey("3h2m1s", t, func() {
		result, err := ParseDuration("3h2m1s")
		So(err, ShouldBeNil)
		So(result, ShouldEqual, time.Duration(time.Hour*3+time.Minute*2+time.Second))
	})
}

func TestToStringArray(t *testing.T) {
	Convey("convertes", t, func() {
		mp := make(map[string]struct{})
		mp["one"] = struct{}{}
		mp["two"] = struct{}{}

		arr := ToStringArray(mp)

		So(arr, ShouldResemble, []string{"one", "two"})
	})
}

func TestChannelDuplication(t *testing.T) {
	Convey("duplicates", t, func() {
		ch := make(chan *hungryfox.Diff)
		item := &hungryfox.Diff{}

		ch1, ch2 := Duplicate(ch, 1)
		ch <- item
		a, b := <-ch1, <-ch2

		So(a, ShouldEqual, item)
		So(b, ShouldEqual, item)
	})
	Convey("closes channels", t, func() {
		ch := make(chan *hungryfox.Diff)

		ch1, ch2 := Duplicate(ch, 1)
		close(ch)
		_, ok1 := <-ch1
		_, ok2 := <-ch2

		So(ok1, ShouldBeFalse)
		So(ok2, ShouldBeFalse)
	})
	Convey("Shouldn't block when buffer isn't filled", t, func() {
		hasBlocked := duplicateChanAndWriteValues(3, 3)
		So(hasBlocked, ShouldBeFalse)
	})
	Convey("May block on writing to ch1 when buffer fills", t, func() {
		hasBlocked := duplicateChanAndWriteValues(3, 4)
		So(hasBlocked, ShouldBeTrue)
	})
}

func duplicateChanAndWriteValues(buffer, writesCount int) (hasBlocked bool) {
	ch := make(chan *hungryfox.Diff)
	timerCh := make(chan struct{})

	_, ch2 := Duplicate(ch, buffer)

	go func() {
		item := &hungryfox.Diff{}
		for i := 0; i < writesCount; i++ {
			ch <- item
		}
		close(ch)
	}()
	go func() {
		time.Sleep(300)
		timerCh <- struct{}{}
	}()

	for {
		select {
		case _, ok := <-ch2:
			if !ok {
				return false
			}
		case <-timerCh:
			return true
		}
	}
}
