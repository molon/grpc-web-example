package main

import (
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	_ = pflag.String("grpc.address", "127.0.0.1:8080", "grpc address")
	_ = pflag.Bool("grpc.reflection", true, "")
	_ = pflag.String("http.address", "127.0.0.1:8080", "http address(grpcweb)")
	_ = pflag.Bool("tls", false, "whether to run TLS server")
	_ = pflag.String("tls.cert.file", "./localhost.crt", "")
	_ = pflag.String("tls.key.file", "./localhost.key", "")
)

func init() {
	pflag.Parse()
	_ = viper.BindPFlags(pflag.CommandLine)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}
