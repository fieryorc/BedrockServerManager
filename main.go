package main

import (
	"context"
	"flag"
	"os"

	"github.com/fieryorc/BedrockServerManager/svrmgr"
	"github.com/golang/glog"
)

func main() {
	flag.Parse()
	glog.Error()
	mgr := svrmgr.NewServerManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := mgr.Process(ctx, os.Args)
	if err != nil {
		panic(err)
	}
}
