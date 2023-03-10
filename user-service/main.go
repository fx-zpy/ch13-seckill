package main

import (
	"context"
	"flag"
	"fmt"
	localconfig "github.com/fx-zpy/SecKillGolang/oauth-service/config"
	"github.com/fx-zpy/SecKillGolang/pb"
	"github.com/fx-zpy/SecKillGolang/pkg/bootstrap"
	conf "github.com/fx-zpy/SecKillGolang/pkg/config"
	register "github.com/fx-zpy/SecKillGolang/pkg/discover"
	"github.com/fx-zpy/SecKillGolang/pkg/mysql"
	"github.com/fx-zpy/SecKillGolang/user-service/endpoint"
	"github.com/fx-zpy/SecKillGolang/user-service/plugins"
	"github.com/fx-zpy/SecKillGolang/user-service/service"
	"github.com/fx-zpy/SecKillGolang/user-service/transport"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	kitzipkin "github.com/go-kit/kit/tracing/zipkin"

	"github.com/openzipkin/zipkin-go/propagation/b3"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	var (
		servicePort = flag.String("service.port", bootstrap.HttpConfig.Port, "service port")
		grpcAddr    = flag.String("grpc", ":9008", "gRPC listen address.")
	)

	flag.Parse()

	ctx := context.Background()
	errChan := make(chan error)

	fieldKeys := []string{"method"}
	requestCount := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "aoho",
		Subsystem: "user_service",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)

	requestLatency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "aoho",
		Subsystem: "user_service",
		Name:      "request_latency",
		Help:      "Total duration of requests in microseconds.",
	}, fieldKeys)
	ratebucket := rate.NewLimiter(rate.Every(time.Second*1), 100)

	var svc service.Service
	svc = service.UserService{}

	// add logging middleware
	svc = plugins.LoggingMiddleware(localconfig.Logger)(svc)
	svc = plugins.Metrics(requestCount, requestLatency)(svc)

	userPoint := endpoint.MakeUserEndpoint(svc)
	userPoint = plugins.NewTokenBucketLimitterWithBuildIn(ratebucket)(userPoint)
	userPoint = kitzipkin.TraceEndpoint(localconfig.ZipkinTracer, "user-endpoint")(userPoint)

	//?????????????????????Endpoint
	healthEndpoint := endpoint.MakeHealthCheckEndpoint(svc)
	healthEndpoint = kitzipkin.TraceEndpoint(localconfig.ZipkinTracer, "health-endpoint")(healthEndpoint)

	endpts := endpoint.UserEndpoints{
		UserEndpoint:        userPoint,
		HealthCheckEndpoint: healthEndpoint,
	}

	//??????http.Handler
	r := transport.MakeHttpHandler(ctx, endpts, localconfig.ZipkinTracer, localconfig.Logger)

	//http server
	go func() {
		fmt.Println("Http Server start at port:" + *servicePort)
		mysql.InitMysql(conf.MysqlConfig.Host, conf.MysqlConfig.Port, conf.MysqlConfig.User, conf.MysqlConfig.Pwd, conf.MysqlConfig.Db)
		//?????????????????????
		register.Register()
		handler := r
		errChan <- http.ListenAndServe(":"+*servicePort, handler)
	}()
	//grpc server
	go func() {
		fmt.Println("grpc Server start at port" + *grpcAddr)
		listener, err := net.Listen("tcp", *grpcAddr)
		if err != nil {
			errChan <- err
			return
		}
		serverTracer := kitzipkin.GRPCServerTrace(localconfig.ZipkinTracer, kitzipkin.Name("grpc-transport"))
		tr := localconfig.ZipkinTracer
		md := metadata.MD{}
		parentSpan := tr.StartSpan("test")

		b3.InjectGRPC(&md)(parentSpan.Context())

		ctx := metadata.NewIncomingContext(context.Background(), md)
		handler := transport.NewGRPCServer(ctx, endpts, serverTracer)
		gRPCServer := grpc.NewServer()
		pb.RegisterUserServiceServer(gRPCServer, handler)
		errChan <- gRPCServer.Serve(listener)
	}()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()

	error := <-errChan
	//????????????????????????
	register.Deregister()
	fmt.Println(error)
}
