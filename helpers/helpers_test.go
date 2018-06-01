package helpers

import (
	"testing"
	"time"

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
