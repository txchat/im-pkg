package trace

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	clusterSarama "github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"gopkg.in/Shopify/sarama.v1"
)

const Topic = "trace4kafka"

var (
	Producer sarama.SyncProducer

	Brokers = []string{"172.16.101.107:9092"}
)

func TestMain(m *testing.M) {
	//init trace
	cfg := config.Configuration{
		ServiceName: "nimaa",
		Gen128Bit:   true,
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: "172.16.101.130:6831",
		},
	}
	tracer, _, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	//defer closer.Close()
	//不然后续不会有Jaeger实例
	opentracing.SetGlobalTracer(tracer)

	//init kafka
	kc := sarama.NewConfig()
	kc.Producer.RequiredAcks = sarama.WaitForAll // Wait for all in-sync replicas to ack the message
	kc.Producer.Retry.Max = 10                   // Retry up to 10 times to produce the message
	kc.Producer.Return.Successes = true
	kc.Version = sarama.V0_11_0_2
	Producer, err = sarama.NewSyncProducer(Brokers, kc)
	if err != nil {
		panic(err)
	}
	exitCode := m.Run()
	time.Sleep(time.Second * 10)
	os.Exit(exitCode)
}

func TestLocalTransform(t *testing.T) {
	initChan := make(chan bool)
	down := make(chan bool)

	go func() {
		//consumer
		c := cluster.NewConfig()
		c.Consumer.Return.Errors = true
		c.Group.Return.Notifications = true
		c.Version = clusterSarama.V0_11_0_2

		Consumer, err := cluster.NewConsumer(Brokers, fmt.Sprintf("%s-group2", Topic), []string{Topic}, c)
		if err != nil {
			t.Error(err)
			return
		}
		//
		num := 0
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*60))
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				t.Error("time out quit")
				down <- true
			case err := <-Consumer.Errors():
				t.Error(fmt.Sprintf("consumer error: %v", err))
			case n := <-Consumer.Notifications():
				num++
				log.Info().Interface("number", n).Msg("consumer rebalanced")
				t.Log(fmt.Sprintf("consumer rebalanced: %v", n))
				if num > 1 {
					initChan <- true
				}
			case msg, ok := <-Consumer.Messages():
				t.Log("rev msg")
				if !ok {
					t.Error(fmt.Sprintf("consume not ok"))
					down <- true
				}
				//do
				tracer := opentracing.GlobalTracer()
				spanContext, err := ExtractMQHeader(tracer, msg)
				if err != nil {
					t.Error(err)
				}
				span := tracer.StartSpan("Kafka Subscribe:"+msg.Topic, opentracing.ChildOf(spanContext))
				span.Finish()
				//return opentracing.ContextWithSpan(context.Background(), span)

				//span := opentracing.SpanFromContext(ctx)
				//if span != nil {
				//	defer span.Finish()
				//}
				Consumer.MarkOffset(msg, "")
				t.Log("success")
				down <- true
			}
		}
	}()

	<-initChan
	time.Sleep(time.Second * 2)
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan("publish")
	ctx := opentracing.ContextWithSpan(context.Background(), span)
	//producer
	m := &sarama.ProducerMessage{
		Key:   sarama.StringEncoder(uuid.New().String()),
		Topic: Topic,
		Value: sarama.ByteEncoder([]byte("test")),
	}
	InjectMQHeader(tracer, span.Context(), ctx, m)
	partition, offset, err := Producer.SendMessage(m)
	if err != nil {
		t.Error(err)
		return
	}
	span.Finish()
	t.Log("publish", partition, offset)
	<-down
}
