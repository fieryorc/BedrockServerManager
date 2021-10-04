package main

import (
	"flag"
	"os"

	"github.com/fieryorc/BedrockServerManager/svrmgr"
)

func main() {
	flag.Parse()
	mgr := svrmgr.NewServerManager()
	err := mgr.Process(os.Args)
	if err != nil {
		panic(err)
	}
}
