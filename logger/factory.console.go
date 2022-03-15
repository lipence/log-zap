package logger

import (
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var mainPath string

func init() {
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		mainPath = buildInfo.Main.Path + "/"
	}
}

func jsonEncoder() zapcore.Encoder {
	return zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})
}

func readableEncoder(enableColor bool) zapcore.Encoder {
	var LevelEncoder zapcore.LevelEncoder
	var NameEncoder zapcore.NameEncoder
	var RFC3339TimeEncoder zapcore.TimeEncoder
	var ShortCallerEncoder zapcore.CallerEncoder
	if enableColor {
		color.NoColor = false
		var BoldCyan = color.New(color.Bold, color.FgCyan)
		var BoldHiBlue = color.New(color.Bold, color.FgHiBlue)
		var HiBlack = color.New(color.FgHiBlack)
		var White = color.New(color.FgWhite)
		var yellow = color.New(color.FgYellow)
		var textHiBlackT = HiBlack.Sprint("T")
		var textYellowLSB = yellow.Sprint("[")
		var textYellowRSB = yellow.Sprint("]")
		LevelEncoder = zapcore.CapitalColorLevelEncoder
		NameEncoder = func(s string, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(fmt.Sprintf("%s%s%s", textYellowLSB, s, textYellowRSB))
		}
		RFC3339TimeEncoder = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			str := BoldCyan.Sprint(t.Format("2006-01-02")) + textHiBlackT +
				BoldHiBlue.Sprint(t.Format("15:04:05")) +
				HiBlack.Sprint(t.Format(".000000Z0700"))
			enc.AppendString(str)
		}
		ShortCallerEncoder = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(White.Sprintf("%s:%d", strings.TrimPrefix(caller.File, mainPath), caller.Line))
		}
	} else {
		NameEncoder = func(s string, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(fmt.Sprintf("[%s]", s))
		}
		LevelEncoder = zapcore.CapitalLevelEncoder
		RFC3339TimeEncoder = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02T15:04:05.000000Z0700"))
		}
		ShortCallerEncoder = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(fmt.Sprintf("%s:%d", strings.TrimPrefix(caller.File, mainPath), caller.Line))
		}
	}

	return zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		CallerKey:      "C",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    LevelEncoder,
		EncodeName:     NameEncoder,
		EncodeTime:     RFC3339TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   ShortCallerEncoder,
	})
}

func consoleCoreFactory(opts *Options) (cores []zapcore.Core, closers []func(), err error) {
	var consoleEncoderObj = readableEncoder(opts.Mode == ModeDevelop)
	var ConsoleInfoCloser, ConsoleErrorCloser func()
	var ConsoleInfoSyncer, ConsoleErrorSyncer zapcore.WriteSyncer
	if ConsoleInfoSyncer, ConsoleInfoCloser, err = zap.Open("stdout"); err != nil {
		return nil, nil, fmt.Errorf("cant init logger console writeSyncer: stdout: %w", err)
	}
	if ConsoleErrorSyncer, ConsoleErrorCloser, err = zap.Open("stderr"); err != nil {
		return nil, nil, fmt.Errorf("cant init logger console writeSyncer: stderr: %w", err)
	}
	var stdoutMinLevel zapcore.Level
	if opts.Mode == ModeProduct {
		stdoutMinLevel = zapcore.InfoLevel
	} else {
		stdoutMinLevel = zapcore.DebugLevel
	}
	cores = append(cores, zapcore.NewCore(consoleEncoderObj, ConsoleInfoSyncer,
		zap.LevelEnablerFunc(func(lvl zapcore.Level) bool { return lvl >= stdoutMinLevel && lvl <= zapcore.InfoLevel }),
	), zapcore.NewCore(consoleEncoderObj, ConsoleErrorSyncer,
		zap.LevelEnablerFunc(func(lvl zapcore.Level) bool { return lvl > zapcore.InfoLevel }),
	))
	closers = []func(){
		ConsoleInfoCloser,
		ConsoleErrorCloser,
	}
	return cores, closers, nil
}
