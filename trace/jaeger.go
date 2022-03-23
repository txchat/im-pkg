package trace

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
	"io"
)

const TraceIDLogKey = "traceID"

// Init returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
func Init(defaultSrvName string, cfg config.Configuration, options ...config.Option) (opentracing.Tracer, io.Closer) {
	if cfg.ServiceName == "" {
		cfg.ServiceName = defaultSrvName
	}
	tracer, closer, err := cfg.NewTracer(options...)
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}
