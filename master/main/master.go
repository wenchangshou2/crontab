package main

import (
	"flag"
	"fmt"
	"github.com/wenchangshou2/crontab/master"
	"runtime"
	"time"
)

var (
	configFile string
)

func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}
func initArgs() {
	flag.StringVar(&configFile, "config", "./master.json", "传入master.json")
	flag.Parse()
}
func main() {
	var (
		err error
	)
	initArgs()
	initEnv()
	if err = master.InitConfig(configFile); err != nil {
		goto ERR
	}
	if err = master.InitLogSink(); err != nil {
		goto ERR
	}
	if err = master.InitJobMgr(); err != nil {
		goto ERR
	}
	if err = master.InitApiServer(); err != nil {
		goto ERR
	}
	for {
		time.Sleep(1 * time.Second)
	}
	return
ERR:
	fmt.Println(err)

}
