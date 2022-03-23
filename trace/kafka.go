package trace

import (
	"context"
	clusterSarama "github.com/Shopify/sarama"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/metadata"
	"gopkg.in/Shopify/sarama.v1"
)

func InjectMQHeader(tracer opentracing.Tracer, sm opentracing.SpanContext, ctx context.Context, msg *sarama.ProducerMessage) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}
	carrier := TextMapRW{md}
	err := tracer.Inject(sm, opentracing.TextMap, carrier)
	if err != nil {
		// maybe rpc client not transform trace instance, it should be work continue
	}
	msg.Headers = NewOutgoingHeaders(msg.Headers, md)
}

func ExtractMQHeader(tracer opentracing.Tracer, msg *clusterSarama.ConsumerMessage) (opentracing.SpanContext, error) {
	md := FromIncomingHeaders(msg.Headers)
	carrier := TextMapRW{md}
	return tracer.Extract(opentracing.TextMap, carrier)
}

//将map[string][]string数据格式转换为[]struct{[]byte, []byte}格式
func NewOutgoingHeaders(h []sarama.RecordHeader, md metadata.MD) []sarama.RecordHeader {
	var headers []sarama.RecordHeader
	copy(headers, h)
	for key, val := range md {
		for _, v := range val {
			header := sarama.RecordHeader{
				Key:   []byte(key),
				Value: []byte(v),
			}
			headers = append(headers, header)
		}
	}
	return headers
}

func FromIncomingHeaders(headers []*clusterSarama.RecordHeader) metadata.MD {
	md := metadata.MD{}
	for _, header := range headers {
		md.Append(string(header.Key), string(header.Value))
	}
	return md
}
