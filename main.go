package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	echov1 "github.com/molon/grpc-web-example/gen/go/grpc/gateway/testing"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		log.Printf("%+v", err)
	}
}

func run() error {
	grpcServer := grpc.NewServer()
	echov1.RegisterEchoServiceServer(grpcServer, &server{})
	wrappedGrpc := grpcweb.WrapServer(grpcServer,
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
		// grpcweb.WithCorsForRegisteredEndpointsOnly(false),
	)
	serveMux := http.NewServeMux()
	serveMux.Handle("/", wrappedGrpc)
	httpServer := &http.Server{
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		Handler:      serveMux,
	}
	httpListener, err := net.Listen("tcp", viper.GetString("grpcweb.address"))
	if err != nil {
		return err
	}
	doneC := make(chan error)
	go func() {
		log.Printf("listening on: %v", httpListener.Addr().String())
		if err := httpServer.Serve(httpListener); err != nil {
			doneC <- fmt.Errorf("server error: %w", err)
		}
	}()

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigC
		doneC <- nil
	}()
	if err := <-doneC; err != nil {
		return err
	}
	return nil
}

type server struct {
	echov1.UnimplementedEchoServiceServer
}

func (*server) Echo(ctx context.Context, in *echov1.EchoRequest) (*echov1.EchoResponse, error) {
	return &echov1.EchoResponse{
		Message: in.GetMessage(),
	}, nil
}

func (*server) ServerStreamingEcho(in *echov1.ServerStreamingEchoRequest, stream echov1.EchoService_ServerStreamingEchoServer) error {
	ticker := time.NewTicker(time.Duration(in.GetMessageInterval()) * time.Millisecond)
	defer ticker.Stop()

	cnt := 0
	for range ticker.C {
		err := stream.Send(&echov1.ServerStreamingEchoResponse{
			Message: in.GetMessage(),
		})
		if err != nil {
			return err
		}
		cnt++
		if cnt >= int(in.GetMessageCount()) {
			break
		}
	}
	return nil
}
