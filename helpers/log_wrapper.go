package helpers

import (
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/rs/zerolog"
)

type zerologGokitLogger func(msg string, keyvalues ...interface{})

func (l zerologGokitLogger) Log(kv ...interface{}) error {
	l("", kv...)
	return nil
}

func WrapDebug(logger zerolog.Logger) log.Logger {
	var wrapper zerologGokitLogger
	wrapper = func(msg string, keyvalues ...interface{}) {
		kv := make([]interface{}, len(keyvalues))
		copy(kv, keyvalues)
		l := logger.Debug()
		var key string
		for i, item := range kv {
			if i%2 == 0 {
				key = fmt.Sprintf("%v", item)
			} else {
				l.Interface(key, item)
			}
		}
		l.Msg("")
	}
	return wrapper
}
