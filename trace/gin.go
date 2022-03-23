package trace

import (
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

func OpenTracingServerMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		tracer := opentracing.GlobalTracer()
		// 继承别的进程传递过来的上下文
		spanContext, _ := tracer.Extract(opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(c.Request.Header))

		span := tracer.StartSpan("GIN Server:"+c.Request.RequestURI, opentracing.ChildOf(spanContext))
		defer span.Finish()

		ctx := opentracing.ContextWithSpan(c.Request.Context(), span)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		//设置 http请求结果
		ext.HTTPStatusCode.Set(span, uint16(c.Writer.Status()))
	}
}
