// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Sample contains a simple http server that exports to the OpenTelemetry agent.
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"google.golang.org/grpc"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// SpanNameVariety SpanName 发散程度（多少个不同值）
const SpanNameVariety = 1000

// AttrValueVariety 属性值发散程度（多少个不同值）
const AttrValueVariety = 10000

// AttrMaxLen AttrMinLen tag value 长度范围
const AttrMaxLen = 10000
const AttrMinLen = 1000

// SpanNameMaxLen SpanNameMinLen span name 长度范围
const SpanNameMaxLen = 64
const SpanNameMinLen = 32

const ServerServiceName = "otlp-server"
const TraceInstrumentationName = "otlp-demo-tracer"
const otelAgentAddr = "tracing-analysis-dc-hz.aliyuncs.com:8090"
const xtraceToken = "<你的阿里云 grpc token>"

var avaAttrValue = [AttrValueVariety]string{}
var avaSpanName = [SpanNameVariety]string{}

// initProvider 初始化 opentelemetry 配置。
//
// Initializes an OTLP exporter, and configures the corresponding trace and
// metric providers.
func initProvider() func() {
	ctx := context.Background()

	// 使用 gRPC 连接阿里云

	headers := map[string]string{"Authentication": xtraceToken}
	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelAgentAddr),
		otlptracegrpc.WithHeaders(headers), // 鉴权信息
		otlptracegrpc.WithDialOption(grpc.WithBlock()))

	// 使用 http 连接阿里云
	//traceClientHttp := otlptracehttp.NewClient(
	//	otlptracehttp.WithEndpoint("tracing-analysis-dc-hz.aliyuncs.com"),
	//	otlptracehttp.WithURLPath("<你的阿里云 HTTP 接入点>"),
	//	otlptracehttp.WithInsecure())
	//otlptracehttp.WithCompression(1)

	// 创建和阿里云链路服务的连接
	log.Println("start to connect to server")
	traceExp, err := otlptrace.New(ctx, traceClient)
	handleErr(err, "Failed to create the collector trace exporter")

	log.Println("trace new finish")

	// 配置 opentelemetry 基本信息
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			// 这是在阿里云会显示的服务名
			semconv.ServiceNameKey.String(ServerServiceName),
		),
	)
	handleErr(err, "failed to create resource")
	log.Println("resource new finish")

	// 配置追踪数据和数据出口（阿里云）之间的联系
	bsp := sdktrace.NewBatchSpanProcessor(traceExp) // 将数据出口绑定到 SpanProcessor
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp), // 将 SpanProcessor 绑定到 TracerProvider
	)

	// set global propagator to traceContext (the default is no-op).
	// 配置 opentelemetry 全局变量
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider) // 设定全局 TracerProvider

	return func() {
		cxt, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := traceExp.Shutdown(cxt); err != nil {
			otel.Handle(err)
		}
	}
}

// handleErr 输出错误信息，希望错误体现在阿里云链路追踪上，见 https://opentelemetry.io/docs/instrumentation/go/getting-started/#bonus-errors
// 一般用以下两个方法标记错误
// span.RecordError(err)
// span.SetStatus(codes.Error, err.Error())
func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

// initTraceDemoData 初始化测试用的随机数据
func initTraceDemoData() {
	for i := 0; i < len(avaAttrValue); i++ {
		//avaAttrValue[i] = common.GenStrWithRandomLen(AttrMinLen, AttrMaxLen)
		avaAttrValue[i] = "AttrValue " + strconv.Itoa(i)
	}

	for i := 0; i < len(avaSpanName); i++ {
		// avaSpanName[i] = common.GenStrWithRandomLen(SpanNameMinLen, SpanNameMaxLen)
		avaSpanName[i] = "SpanName " + strconv.Itoa(i)
	}
}

func main() {

	// 初始化 opentelemetry 和链路追踪后端（阿里云）进行连接
	shutdown := initProvider()
	defer shutdown()

	//meter := global.Meter("demo-server-meter")
	serverAttribute := attribute.String("server-attribute", "foo")
	fmt.Println("start to gen chars for trace data")
	// 随机生成用于测试的数据
	initTraceDemoData()
	fmt.Println("gen trace data done")
	// 声明链路追踪的名字
	tracer := otel.Tracer(TraceInstrumentationName)

	// create a handler wrapped in OpenTelemetry instrumentation
	// 建立一个 http 请求的处理函数
	// 在这个处理函数中，先休眠随机的时间
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		//  random sleep to simulate latency
		//var sleep int64
		//switch modulus := time.Now().Unix() % 5; modulus {
		//case 0:
		//	sleep = rng.Int63n(2000)
		//case 1:
		//	sleep = rng.Int63n(15)
		//case 2:
		//	sleep = rng.Int63n(917)
		//case 3:
		//	sleep = rng.Int63n(87)
		//case 4:
		//	sleep = rng.Int63n(1173)
		//}
		ctx := req.Context()
		span := trace.SpanFromContext(ctx)  // 获得自己在当前链路中的节点信息
		span.SetAttributes(serverAttribute) // 设置自己的节点属性

		//actionChild(tracer, ctx, sleep) // 生成自己的子 span
		actionChild(tracer, ctx, 0) // 生成自己的子 span
		connectFlask(tracer, ctx)   // 向 Python Flask 客户端发送请求
		connectPhp(tracer, ctx)     // 向 PHP 客户端发送请求
		connectSpring(tracer, ctx)  // 向 Java Spring 客户端发送请求

		_, err := w.Write([]byte("Hello World"))
		if err != nil {
			fmt.Println(err)
			return
		}
	})
	wrappedHandler := otelhttp.NewHandler(handler, "/hello")

	// serve up the wrapped handler
	http.Handle("/hello", wrappedHandler)
	err := http.ListenAndServe(":7080", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
}

// actionChild 生成一个链路事件，即一个子 span
func actionChild(tracer trace.Tracer, ctx context.Context, sleep int64) {
	_, subSpan := tracer.Start(ctx, "back-end subSpan")
	defer subSpan.End()
	// time.Sleep(time.Duration(sleep) * time.Millisecond)
	// 此处用于测试 span 错误
	errTest := fmt.Errorf("测试：span 发生错误")
	// fmt.Println("测试：span 发生错误")
	subSpan.RecordError(errTest)
	subSpan.SetStatus(codes.Error, errTest.Error())
	// subSpan.SetStatus(codes.Ok, "success")
	serverAttribute := attribute.String("attr1", "attr for test")
	subSpan.SetAttributes(serverAttribute)
}

func connectSpring(tracer trace.Tracer, ctx context.Context) {
	_, subSpan := tracer.Start(ctx, "backend-connect-spring")
	defer subSpan.End()

	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:5638/test", nil)
	if err != nil {
		handleErr(err, "failed to http request")
	}
	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
	} else {
		err := res.Body.Close()
		if err != nil {
			return
		}
	}

	subSpan.SetStatus(codes.Ok, "success")
	serverAttribute := attribute.String("attr_test", "Go_to_Java")
	subSpan.SetAttributes(serverAttribute)
}

func connectPhp(tracer trace.Tracer, ctx context.Context) {
	_, subSpan := tracer.Start(ctx, "backend-connect-php")
	defer subSpan.End()

	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8083/", nil)
	if err != nil {
		handleErr(err, "failed to http request")
	}
	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
	} else {
		err := res.Body.Close()
		if err != nil {
			return
		}
	}

	subSpan.SetStatus(codes.Ok, "success")
	serverAttribute := attribute.String("attr_test", "Go_to_Php")
	subSpan.SetAttributes(serverAttribute)
}

func connectFlask(tracer trace.Tracer, ctx context.Context) {
	_, subSpan := tracer.Start(ctx, "backend-connect-flask")
	defer subSpan.End()

	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:5000/test", nil)
	if err != nil {
		handleErr(err, "failed to http request")
	}
	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
	} else {
		err := res.Body.Close()
		if err != nil {
			return
		}
	}

	subSpan.SetStatus(codes.Ok, "success")
	serverAttribute := attribute.String("attr_test", "Go_to_Python")
	subSpan.SetAttributes(serverAttribute)
}

func getRandomAttrValue() string {
	n := rand.Int63n(int64(len(avaAttrValue)))
	return avaAttrValue[n]
}

func getRandomSpanName() string {
	n := rand.Int63n(int64(len(avaSpanName)))
	return avaSpanName[n]
}
