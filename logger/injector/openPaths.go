package injector

import (
	_ "unsafe"

	"go.uber.org/zap/zapcore"
)

//go:linkname OpenPaths go.uber.org/zap.open
func OpenPaths(paths []string) ([]zapcore.WriteSyncer, func(), error)
