package main

import (
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	_ = pflag.String("grpcweb.address", "127.0.0.1:8080", "grpcweb address")
)

func init() {
	pflag.Parse()
	_ = viper.BindPFlags(pflag.CommandLine)

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}
