package trace

import (
	"fmt"
	"go.uber.org/zap"
	"runtime"
)

func trace() (file string, function string, line int) {
	pc := make([]uintptr, 10)
	n := runtime.Callers(3, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return frame.File, frame.Function, frame.Line
}

func WrapMsg(str string) string {
	_, function, line := trace()
	return fmt.Sprintf("%s:%d: %s", function, line, str)
}

func WrapError(err error) error {
	_, function, line := trace()
	return fmt.Errorf("%s:%d %w", function, line, err)
}

func AsZapError(err error) zap.Field {
	return zap.Error(WrapError(err))
}
