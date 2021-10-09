package main

import (
	"flag"
	"os"

	"github.com/fieryorc/BedrockServerManager/svrmgr"
	"github.com/golang/glog"
)

func main() {
	flag.Parse()
	glog.Error()
	mgr := svrmgr.NewServerManager()
	err := mgr.Process(os.Args)
	if err != nil {
		panic(err)
	}
}
