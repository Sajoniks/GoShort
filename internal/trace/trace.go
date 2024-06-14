package trace

import (
	"errors"
	"fmt"
	"runtime"
)

func trace() (file string, function string, line int) {
	pc := make([]uintptr, 10)
	n := runtime.Callers(3, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return frame.File, frame.Function, frame.Line
}

func WrapError(err error) error {
	_, function, line := trace()
	return errors.Join(err, fmt.Errorf("%s:%d", function, line))
}
