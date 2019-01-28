package main

import (
	"flag"
	"fmt"
	"github.com/wenchangshou2/crontab/worker"
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
	flag.StringVar(&configFile, "config", "./worker.json", "worker.json")
	flag.Parse()
}
func main() {
	var (
		err error
	)
	initArgs()
	initEnv()
	if err = worker.InitExecutor(); err != nil {
		goto ERR
	}
	if err = worker.InitConfig(configFile); err != nil {
		goto ERR
	}
	//服务注册
	if err = worker.InitRegister(); err != nil {
		goto ERR
	}
	//启动日志协程
	if err = worker.InitLogSink(); err != nil {
		goto ERR
	}
	if err = worker.InitScheduler(); err != nil {
		goto ERR
	}
	if err = worker.InitJobMgr(); err != nil {
		goto ERR
	}

	for {
		time.Sleep(1 * time.Second)
	}
	return
ERR:
	fmt.Println(err)

}
