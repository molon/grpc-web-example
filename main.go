package main

import (
	"context"
	"crypto/tls"
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
	"github.com/mwitkow/go-conntrack/connhelpers"
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
	)
	httpServer := &http.Server{
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if wrappedGrpc.IsGrpcWebRequest(req) {
				wrappedGrpc.ServeHTTP(resp, req)
				return
			}
			http.DefaultServeMux.ServeHTTP(resp, req)
		}),
	}

	httpListener, err := net.Listen("tcp", viper.GetString("grpcweb.address"))
	if err != nil {
		return err
	}

	if viper.GetBool("tls") {
		tlsConfig, err := buildServerTlsOrFail(viper.GetString("tls.cert.file"), viper.GetString("tls.key.file"))
		if err != nil {
			return err
		}
		httpListener = tls.NewListener(httpListener, tlsConfig)
	}

	doneC := make(chan error, 1)
	go func() {
		if viper.GetBool("tls") {
			log.Printf("listening tls on: %v", httpListener.Addr().String())
		} else {
			log.Printf("listening on: %v", httpListener.Addr().String())
		}
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

func buildServerTlsOrFail(certFile string, keyFile string) (*tls.Config, error) {
	tlsConfig, err := connhelpers.TlsConfigForServerCerts(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed reading TLS server keys: %w", err)
	}
	tlsConfig.MinVersion = tls.VersionTLS12
	tlsConfig.ClientAuth = tls.NoClientCert
	tlsConfig, err = connhelpers.TlsConfigWithHttp2Enabled(tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("can't configure h2 handling: %w", err)
	}
	return tlsConfig, nil
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
