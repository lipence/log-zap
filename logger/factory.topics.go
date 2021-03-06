package logger

import (
	"fmt"
	"strings"
	_ "unsafe"

	"go.uber.org/zap"
	_ "go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/lipence/log-zap/logger/injector"
)

const (
	ZapTopicConfigEnable   = "Enable"
	ZapTopicConfigEntries  = "Entries"
	ZapTopicConfigProvider = "Provider"
)

func WordMeansTrue(text string) bool {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "true", "yes", "y", "on", "1":
		return true
	default:
		return false
	}
}

type topicURLGenerator interface {
	Provider() string
	Generate(argStore func(string) (string, bool)) (string, error)
}

type paramStoreProxy struct {
	opts   *Options
	entry  string
	prefix string
}

func (p *paramStoreProxy) Get(key string) (string, bool) {
	return p.opts.ParamStore.Get(p.wrap(key))
}

func (p *paramStoreProxy) wrap(key string) string {
	var sep string
	if p.opts.ParamSepStr != "" {
		sep = p.opts.ParamSepStr
	} else {
		sep = "_"
	}
	if p.prefix != "" {
		key = p.prefix + sep + key
	}
	if p.entry != "" {
		key = p.entry + sep + key
	}
	return key
}

func createTopicCore(prefix string, provider string, opts *Options) (core zapcore.Core, closer func(), err error) {
	var hijacker injector.CoreHijacker
	var writeSyncer zapcore.WriteSyncer
	var infoURL string
	{
		var generator topicURLGenerator
		// todo migrate to generic array filter
		for _, _generator := range opts.topicHandlers {
			if strings.EqualFold(_generator.Provider(), provider) {
				generator = _generator
				break
			}
		}
		if generator == nil {
			return nil, nil, fmt.Errorf("undefined topic provider `%s`", provider)
		} else if infoURL, err = generator.Generate(
			(&paramStoreProxy{opts: opts, entry: opts.ParamEntry, prefix: prefix}).Get,
		); err != nil {
			return nil, nil, err
		}
	}
	if infoURL != "" {
		var _syncers []zapcore.WriteSyncer
		if _syncers, closer, err = injector.OpenPaths([]string{infoURL}); err != nil {
			return nil, nil, fmt.Errorf(
				"cant init logger Topic writeSyncer(prefix: %s, provider: %s): %w", prefix, provider, err,
			)
		}
		hijacker, _ = (_syncers[0]).(injector.CoreHijacker)
		writeSyncer = zap.CombineWriteSyncers(_syncers...)
	}
	if hijacker != nil {
		core = hijacker.HijackCore()
	} else {
		core = zapcore.NewCore(readableEncoder(false), writeSyncer,
			zap.LevelEnablerFunc(func(lvl zapcore.Level) bool { return true }),
		)
	}
	return core, closer, nil
}

func topicCoreFactory(opts *Options) (cores []zapcore.Core, closers []func(), err error) {
	var argStore = &paramStoreProxy{opts: opts, entry: opts.ParamEntry, prefix: ""}
	if opts.ParamStore == nil {
		return nil, nil, nil
	}
	var topicEntries = map[string]string{}
	// multi topic mode
	if entryVal, ok := argStore.Get(ZapTopicConfigEntries); ok && entryVal != "" {
		for _, prefix := range strings.Split(entryVal, ",") {
			var provider string
			prefix = strings.TrimSpace(prefix)
			var _argStore = &paramStoreProxy{opts: opts, entry: opts.ParamEntry, prefix: prefix}
			if provider, ok = _argStore.Get(ZapTopicConfigProvider); !ok {
				return nil, nil, fmt.Errorf(
					"undefined environment variable `%s`", _argStore.wrap(ZapTopicConfigProvider))
			}
			topicEntries[prefix] = provider
		}
	}
	// single topic mode
	if enableVal, ok := argStore.Get(ZapTopicConfigEnable); ok && WordMeansTrue(enableVal) {
		var provider string
		if provider, ok = argStore.Get(ZapTopicConfigProvider); ok {
			provider = strings.TrimSpace(provider)
			topicEntries[strings.Title(provider)] = provider
		} else {
			return nil, nil, fmt.Errorf(
				"undefined environment variable `%s`", argStore.wrap(ZapTopicConfigProvider))
		}
	}
	// load topic
	for prefix, provider := range topicEntries {
		if topicCores, consoleClosers, _err := createTopicCore(prefix, provider, opts); _err != nil {
			return nil, nil, _err
		} else {
			cores = append(cores, topicCores)
			closers = append(closers, consoleClosers)
		}
	}
	return cores, closers, nil
}
