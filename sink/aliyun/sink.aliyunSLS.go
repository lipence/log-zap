package sink_aliyun

import (
	"fmt"
	"net/url"

	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/aliyun/aliyun-log-go-sdk/producer"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	AliyunSLSSchema                = "aliyun-sls"
	AliyunSLSParamProject          = "project"
	AliyunSLSParamLogStore         = "logStore"
	AliyunSLSParamSchema           = "schema"
	AliyunSLSParamSource           = "source"
	AliyunSLSConfigProject         = "Project"
	AliyunSLSConfigLogStore        = "LogStore"
	AliyunSLSConfigEndpoint        = "Endpoint"
	AliyunSLSConfigAccessKeyID     = "AccessKeyID"
	AliyunSLSConfigAccessKeySecret = "AccessKeySecret" // #nosec G101
)

type aliyunSLSCore struct {
	sink   *aliyunSLSSink
	fields []zapcore.Field
}

func (core *aliyunSLSCore) Enabled(zapcore.Level) bool {
	return true
}

func (core *aliyunSLSCore) With(f []zapcore.Field) zapcore.Core {
	return &aliyunSLSCore{
		sink:   core.sink,
		fields: append(core.fields, f...),
	}
}

func (core *aliyunSLSCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry { // nolint:gocritic
	if core.Enabled(ent.Level) {
		return ce.AddCore(ent, core)
	}
	return ce
}

func (core *aliyunSLSCore) Write(e zapcore.Entry, fields []zapcore.Field) (err error) { // nolint:gocritic
	fields = append(core.fields, fields...)
	enc := zapcore.NewMapObjectEncoder()
	enc.Fields["level"] = e.Level.String()
	enc.Fields["caller"] = e.Caller.FullPath()
	enc.Fields["msg"] = e.Message
	for _, field := range fields {
		field.AddTo(enc)
	}
	var data = make(map[string]string, len(fields))
	for k, v := range enc.Fields {
		data[k] = fmt.Sprint(v)
	}
	return core.sink.write(producer.GenerateLog(uint32(e.Time.Unix()), data), e.LoggerName)
}

func (core *aliyunSLSCore) Sync() error {
	return nil
}

type aliyunSLSSink struct {
	source   string
	project  string
	logStore string
	producer *producer.Producer
}

func (sink *aliyunSLSSink) HijackCore() zapcore.Core {
	return &aliyunSLSCore{sink: sink}
}
func (sink *aliyunSLSSink) write(l *sls.Log, topic string) error {
	if topic == "" {
		topic = "none"
	}
	return sink.producer.SendLog(sink.project, sink.logStore, topic, sink.source, l)
}

func (sink *aliyunSLSSink) Write(_ []byte) (int, error) {
	return 0, fmt.Errorf("use *aliyunSLSCore instead")
}

func (sink *aliyunSLSSink) Sync() error {
	return nil
}

func (sink *aliyunSLSSink) Close() error {
	sink.producer.SafeClose()
	return nil
}

func register(logPath *url.URL) (sink zap.Sink, err error) {
	producerConfig := producer.GetDefaultProducerConfig()
	producerConfig.Endpoint = logPath.Host
	producerConfig.AccessKeyID = logPath.User.Username()
	producerConfig.AccessKeySecret, _ = logPath.User.Password()
	var urlQuery = logPath.Query()
	if schema := urlQuery.Get(AliyunSLSParamSchema); schema != "" {
		producerConfig.Endpoint = fmt.Sprintf("%s://%s", schema, producerConfig.Endpoint)
	}
	var _sink = &aliyunSLSSink{
		source:   urlQuery.Get(AliyunSLSParamSource),
		project:  urlQuery.Get(AliyunSLSParamProject),
		logStore: urlQuery.Get(AliyunSLSParamLogStore),
		producer: producer.InitProducer(producerConfig),
	}
	_sink.producer.Start()
	return _sink, nil
}

func init() {
	if err := zap.RegisterSink(AliyunSLSSchema, register); err != nil {
		panic(fmt.Errorf("cant register aliyun-sls sink: %w", err))
	}
}

type urlGenerator struct {
	topic  string
	source string
}

func (g *urlGenerator) WithTopic(topic string) *urlGenerator {
	return &urlGenerator{
		topic: topic,
	}
}

func (g *urlGenerator) Provider() string {
	if g.topic != "" {
		return g.topic
	}
	return AliyunSLSSchema
}

func (g *urlGenerator) Generate(argStore func(string) (string, bool)) (string, error) {
	var ok bool
	var outputPath = &url.URL{Scheme: AliyunSLSSchema}
	var outputQuery = url.Values{}
	{
		var endpoint string
		if endpoint, ok = argStore(AliyunSLSConfigEndpoint); !ok {
			return "", fmt.Errorf("`Endpoint` not optional")
		}
		if scheme, host, err := g.parseEndpoint(endpoint); err == nil {
			outputPath.Host = host
			outputQuery.Set(AliyunSLSParamSchema, scheme)
		} else {
			return "", err
		}
	}
	{
		var source string
		if source = g.source; source == "" {
			return "", fmt.Errorf("`Source` not optional")
		}
		outputQuery.Set(AliyunSLSParamSource, source)
	}
	{
		var ak, sk string
		if ak, ok = argStore(AliyunSLSConfigAccessKeyID); !ok {
			return "", fmt.Errorf("`AccessKeyID` not optional")
		}
		if sk, ok = argStore(AliyunSLSConfigAccessKeySecret); !ok {
			return "", fmt.Errorf("`AccessKeySecret` not optional")
		}
		outputPath.User = url.UserPassword(ak, sk)
	}
	{
		var project, logStore string
		if project, ok = argStore(AliyunSLSConfigProject); !ok {
			return "", fmt.Errorf("`Project` not optional")

		}
		if logStore, ok = argStore(AliyunSLSConfigLogStore); !ok {
			return "", fmt.Errorf("`LogStore` not optional")
		}
		outputQuery.Set(AliyunSLSParamProject, project)
		outputQuery.Set(AliyunSLSParamLogStore, logStore)
	}
	outputPath.RawQuery = outputQuery.Encode()
	return outputPath.String(), nil
}

func (g *urlGenerator) parseEndpoint(endpoint string) (string, string, error) {
	if endpointURL, err := url.Parse(endpoint); err != nil {
		return "", "", fmt.Errorf("cant parse aliyun-sls endpoint: %w", err)
	} else if endpointURL.Scheme == "" {
		return "http", endpoint, nil
	} else {
		return endpointURL.Scheme, endpointURL.Host, nil
	}
}

func NewURLGenerator(source string) *urlGenerator {
	return &urlGenerator{source: source}
}
