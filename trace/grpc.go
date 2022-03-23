package trace

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Opentracing About
type TextMapRW struct {
	metadata.MD
}

func (t TextMapRW) ForeachKey(handler func(key, val string) error) error {
	for key, val := range t.MD {
		for _, v := range val {
			if err := handler(key, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t TextMapRW) Set(key, val string) {
	t.MD[key] = append(t.MD[key], val)
}

func OpentracingServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	tracer := opentracing.GlobalTracer()
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}
	carrier := TextMapRW{md}
	spanContext, err := tracer.Extract(opentracing.TextMap, carrier)
	if err != nil {
		// maybe rpc client not transform trace instance, it should be work continue
	}
	span := tracer.StartSpan("gRPC Server:"+info.FullMethod, opentracing.ChildOf(spanContext))
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return handler(ctx, req)
}

func OpentracingClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	tracer := opentracing.GlobalTracer()

	span, ctx := opentracing.StartSpanFromContextWithTracer(ctx, tracer, "gRPC Client:"+method)
	defer span.Finish()

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}
	carrier := TextMapRW{md}
	err := tracer.Inject(span.Context(), opentracing.TextMap, carrier)
	if err != nil {
		// maybe rpc client not transform trace instance, it should be work continue
	}
	ctx = metadata.NewOutgoingContext(ctx, md)
	return invoker(ctx, method, req, reply, cc, opts...)
}
