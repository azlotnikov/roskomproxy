package main

import (
	"context"
	"github.com/9seconds/httransform/v2"
	"github.com/9seconds/httransform/v2/dialers"
	"github.com/9seconds/httransform/v2/executor"
	"github.com/9seconds/httransform/v2/layers"
	"github.com/cosiner/flag"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Params struct {
	Cert string `names:"--cert, -c" usage:"certificate file" default:"server.crt"`
	Key  string `names:"--key, -k" usage:"certificate key file" default:"server.key"`
	Port string `names:"--port, -p" usage:"proxy port" default:"8080"`
	Dns  string `names:"--dns, -d" usage:"dns server, if empty the system one will be used"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for range signals {
			cancel()
		}
	}()

	params := &Params{}
	err := flag.Commandline.ParseStruct(params)
	if err != nil {
		panic(err)
	}

	ca, err := ioutil.ReadFile(params.Cert)
	if err != nil {
		panic(err)
	}
	ck, err := ioutil.ReadFile(params.Key)
	if err != nil {
		panic(err)
	}
	opts := httransform.ServerOpts{
		TLSCertCA:     ca,
		TLSPrivateKey: ck,
		TLSSkipVerify: true,
		Layers: []layers.Layer{
			layers.ProxyHeadersLayer{},
			layers.TimeoutLayer{
				Timeout: 3 * time.Minute,
			},
		},
	}

	dialer := NewUTLSDialer(dialers.Opts{
		TLSSkipVerify: opts.GetTLSSkipVerify(),
	}, params.Dns)
	opts.Executor = executor.MakeDefaultExecutor(dialer)

	proxy, err := httransform.NewServer(ctx, opts)
	if err != nil {
		panic(err)
	}

	listener, err := net.Listen("tcp", ":"+params.Port)
	if err != nil {
		panic(err)
	}

	if err := proxy.Serve(listener); err != nil {
		panic(err)
	}
}
