package main

import (
	"fmt"
	"github.com/9seconds/httransform/v2/layers"
	"time"
)

type LogLayerLayer struct {
	Timeout time.Duration
}

func (l LogLayerLayer) OnRequest(ctx *layers.Context) error {
	req := ctx.Request()
	fmt.Printf("=> %s %s\n", req.Header.Method(), req.URI().String())

	return nil
}

func (l LogLayerLayer) OnResponse(ctx *layers.Context, err error) error {
	res := ctx.Response()
	req := ctx.Request()
	fmt.Printf("<= %d %s %s\n", res.StatusCode(), req.Header.Method(), req.URI().String())

	return nil
}
