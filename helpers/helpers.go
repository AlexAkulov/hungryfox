package helpers

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AlexAkulov/hungryfox"
)

func PrettyDuration(d time.Duration) string {
	s := d.Round(time.Second).String()
	if strings.HasSuffix(s, "m0s") {
		s = s[:len(s)-2]
	}
	if strings.HasSuffix(s, "h0m") {
		s = s[:len(s)-2]
	}
	return s
}

func ParseDuration(str string) (time.Duration, error) {
	// durationRegex := regexp.MustCompile(`(?P<years>\d+y(ears?)?)?(?P<months>\d+m(onths?))?(?P<days>\d+d(ays?)?)?(P?<hours>\d+h(ours?)?)?(?P<minutes>\d+m(in(ute)?s?)?)?(?P<seconds>\d+s(ec(ond)?s?)?)?`)
	durationRegex := regexp.MustCompile(`(?P<years>\d+y)?(?P<days>\d+d)?(?P<hours>\d+h)?(?P<minutes>\d+m)?(?P<seconds>\d+s)?`)
	matches := durationRegex.FindStringSubmatch(str)
	years := ParseInt64(matches[1])
	days := ParseInt64(matches[2])
	hours := ParseInt64(matches[3])
	minutes := ParseInt64(matches[4])
	seconds := ParseInt64(matches[5])

	hour := int64(time.Hour)
	minute := int64(time.Minute)
	second := int64(time.Second)
	duration := time.Duration(years*24*365*hour + days*24*hour + hours*hour + minutes*minute + seconds*second)
	return duration, nil
}

func ParseInt64(value string) int64 {
	if len(value) == 0 {
		return 0
	}
	parsed, err := strconv.Atoi(value[:len(value)-1])
	if err != nil {
		return 0
	}
	return int64(parsed)
}

//TODO: pretty fragile, rewrite/get rid of
func Duplicate(channel <-chan *hungryfox.Diff, buffLen int) (<-chan *hungryfox.Diff, <-chan *hungryfox.Diff) {
	ch1, ch2 := make(chan *hungryfox.Diff, buffLen), make(chan *hungryfox.Diff, buffLen)
	go func() {
		for val := range channel {
			ch1 <- val
			ch2 <- val
		}
		close(ch1)
		close(ch2)
	}()
	return ch1, ch2
}

func ToStringArray(mp map[string]struct{}) []string {
	arr := make([]string, len(mp))
	i := 0
	for key, _ := range mp {
		arr[i] = key
		i++
	}
	return arr
}

func RecoverTo(err *error) {
	e := recover()
	if e != nil {
		switch typedErr := e.(type) {
		case error:
			*err = typedErr
		case string:
			*err = errors.New(typedErr)
		default:
			panic(e)
		}
	}
}
