package injector

import "go.uber.org/zap/zapcore"

type CoreHijacker interface {
	HijackCore() zapcore.Core
}
