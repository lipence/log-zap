package logger

import (
	sysLog "log"

	"go.uber.org/zap"

	"github.com/lipence/log"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	depth int
	*zap.SugaredLogger
}

func (l zapLogger) Print(v ...interface{}) {
	l.Info(v...)
}

func (l zapLogger) Printf(format string, v ...interface{}) {
	l.Infof(format, v...)
}

func (l zapLogger) With(v ...interface{}) log.Logger {
	var _logger = l.SugaredLogger
	if l.depth > 0 {
		_logger = _logger.Desugar().WithOptions(zap.AddCallerSkip(l.depth)).Sugar()
	}
	return zapLogger{SugaredLogger: _logger.With(v...)}
}

func (l zapLogger) WithName(name string) log.Logger {
	var _logger = l.SugaredLogger
	if l.depth > 0 {
		_logger = _logger.Desugar().WithOptions(zap.AddCallerSkip(l.depth)).Sugar()
	}
	return zapLogger{SugaredLogger: _logger.Named(name)}
}

func (l zapLogger) StdLogger() *sysLog.Logger {
	return zap.NewStdLog(l.SugaredLogger.Desugar())
}

func New(opts Options) (logger log.Logger, sync func(), err error) {
	if err = opts.SelfCheck(); err != nil {
		return nil, nil, err
	}
	var cores []zapcore.Core
	var syncers []func()
	{ // console logger
		if consoleCores, consoleClosers, _err := consoleCoreFactory(&opts); _err != nil {
			return nil, nil, _err
		} else {
			cores = append(cores, consoleCores...)
			syncers = append(syncers, consoleClosers...)
		}
	}
	{ // topic logger
		if topicCores, topicClosers, _err := topicCoreFactory(&opts); _err != nil {
			return nil, nil, _err
		} else {
			cores = append(cores, topicCores...)
			syncers = append(syncers, topicClosers...)
		}
	}
	var _logger = zap.New(
		zapcore.NewTee(cores...),
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(
			zap.LevelEnablerFunc(func(lvl zapcore.Level) bool { return lvl >= zapcore.DPanicLevel }),
		),
	)
	var syncer = func() {
		_ = _logger.Sync()
		for _, closer := range syncers {
			if closer != nil {
				closer()
			}
		}
	}
	return &zapLogger{SugaredLogger: _logger.Sugar()}, syncer, nil
}
