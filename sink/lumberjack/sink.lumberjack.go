package sink_lumberjack

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"

	"go.uber.org/zap"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	LumberjackSchema           = "lumberjack"
	LumberjackParamPath        = "path"
	LumberjackParamBase        = "base"
	LumberjackParamMaxSize     = "maxSize"
	LumberjackParamMaxBackups  = "maxBackups"
	LumberjackParamMaxAge      = "maxAge"
	LumberjackConfigPath       = "Path"
	LumberjackConfigMaxSize    = "MaxSize"
	LumberjackConfigMaxBackups = "MaxBackups"
	LumberjackConfigMaxAge     = "MaxAge"
)

type lumberjackSink struct {
	*lumberjack.Logger
}

func (lumberjackSink) Sync() error {
	return nil
}

func targetPath(base string, target string) string {
	if filepath.IsAbs(target) {
		return filepath.Clean(target)
	}
	return filepath.Clean(filepath.Join(base, target))
}

func register(logPath *url.URL) (sink zap.Sink, err error) {
	var params = logPath.Query()
	var fileBase, filePath string
	if fileBase = params.Get(LumberjackParamBase); fileBase == "" {
		return nil, fmt.Errorf("undefined arg `%s`", LumberjackParamBase)
	}
	if filePath = params.Get(LumberjackParamPath); filePath == "" {
		return nil, fmt.Errorf("undefined arg `%s`", LumberjackParamPath)
	}
	var maxSize, maxBackups, maxAge int64
	if maxSizeVal := params.Get(LumberjackParamMaxSize); maxSizeVal != "" {
		if maxSize, err = strconv.ParseInt(maxSizeVal, 10, 32); err != nil {
			return nil, fmt.Errorf("cant parse arg `%s`: %w", LumberjackParamMaxSize, err)
		}
	}
	if maxBackupsVal := params.Get(LumberjackParamMaxBackups); maxBackupsVal != "" {
		if maxBackups, err = strconv.ParseInt(maxBackupsVal, 10, 32); err != nil {
			return nil, fmt.Errorf("cant parse arg `%s`: %w", LumberjackParamMaxBackups, err)
		}
	}
	if maxAgeVal := params.Get(LumberjackParamMaxAge); maxAgeVal != "" {
		if maxAge, err = strconv.ParseInt(maxAgeVal, 10, 32); err != nil {
			return nil, fmt.Errorf("cant parse arg `%s`: %w", LumberjackParamMaxAge, err)
		}
	}
	return lumberjackSink{Logger: &lumberjack.Logger{
		Filename:   targetPath(fileBase, filePath),
		MaxSize:    int(maxSize),
		MaxBackups: int(maxBackups),
		MaxAge:     int(maxAge),
	}}, nil
}

func init() {
	if err := zap.RegisterSink(LumberjackSchema, register); err != nil {
		panic(fmt.Errorf("cant register lumberjack sink: %w", err))
	}
}

type urlGenerator struct {
	topic string
	base  string
}

func (g *urlGenerator) WithTopic(topic string) *urlGenerator {
	return &urlGenerator{
		base:  g.base,
		topic: topic,
	}
}

func (g *urlGenerator) Provider() string {
	if g.topic != "" {
		return g.topic
	}
	return LumberjackSchema
}

func (g *urlGenerator) Generate(argStore func(string) (string, bool)) (string, error) {
	var ok bool
	var outputQuery = url.Values{}
	var outputPath = &url.URL{Scheme: LumberjackSchema, Host: "localhost"}
	{
		var fileBase, filePath string
		if fileBase = g.base; fileBase != "" {
			outputQuery.Set(LumberjackParamBase, fileBase)
		} else {
			return "", fmt.Errorf("unspecificed log base path `%s`", LumberjackParamBase)
		}
		if filePath, ok = argStore(LumberjackConfigPath); ok {
			outputQuery.Set(LumberjackParamPath, filePath)
		} else {
			return "", fmt.Errorf("unspecificed log relative path `%s`", LumberjackConfigPath)
		}
	}
	{
		var maxSize, maxBackups, maxAge string
		if maxSize, ok = argStore(LumberjackConfigMaxSize); ok {
			outputQuery.Set(LumberjackParamMaxSize, maxSize)
		}
		if maxBackups, ok = argStore(LumberjackConfigMaxBackups); ok {
			outputQuery.Set(LumberjackParamMaxBackups, maxBackups)
		}
		if maxAge, ok = argStore(LumberjackConfigMaxAge); ok {
			outputQuery.Set(LumberjackParamMaxAge, maxAge)
		}
	}
	outputPath.RawQuery = outputQuery.Encode()
	return outputPath.String(), nil
}

func NewURLGenerator(baseDir string) *urlGenerator {
	return &urlGenerator{
		base: baseDir,
	}
}
