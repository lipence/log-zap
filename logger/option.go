package logger

import "fmt"

type Mode uint8

const (
	ModeDevelop Mode = iota
	ModeTesting
	ModeProduct
)

func (m Mode) Sting() string {
	switch m {
	case ModeDevelop:
		return "develop"
	case ModeTesting:
		return "testing"
	case ModeProduct:
		return "product"
	default:
		return "unknown"
	}
}

type paramStore interface {
	Get(key string) (value string, exist bool)
}

type Options struct {
	Mode          Mode
	ParamEntry    string
	ParamSepStr   string
	ParamStore    paramStore
	EncJSONOnProd bool
	topicHandlers []topicURLGenerator
}

func (o *Options) SelfCheck() error {
	if o.Mode != ModeDevelop && o.Mode != ModeTesting && o.Mode != ModeProduct {
		return fmt.Errorf("invalid enum `mode`: should be ModeDevelop/ModeTesting/ModeProduct")
	}
	return nil
}

func (o *Options) WithTopic(tg topicURLGenerator) error {
	for _, tgN := range o.topicHandlers {
		if tg.Provider() == tgN.Provider() {
			return fmt.Errorf("conflict topic provider: `%s`", tg.Provider())
		}
	}
	o.topicHandlers = append(o.topicHandlers, tg)
	return nil
}
