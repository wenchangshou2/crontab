package master

import (
	"encoding/json"
	"fmt"
	"github.com/wenchangshou2/crontab/common"
	"net"
	"net/http"
	"strconv"
	"time"
)

type ApiServer struct {
	httpServer *http.Server
}

func handleJobSave(resp http.ResponseWriter, req *http.Request) {
	var (
		err     error
		postJob string
		job     common.Job
		oldJob  *common.Job
		bytes   []byte
	)
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	postJob = req.PostForm.Get("job")
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}
	if oldJob, err = G_JobMgr.SaveJob(&job); err != nil {
		goto ERR
	}
	if bytes, err = common.BuildResponse(0, "success", oldJob); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}
func handleJobDelete(resp http.ResponseWriter, req *http.Request) {
	var (
		err    error
		name   string
		oldJob *common.Job
		bytes  []byte
	)
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	name = req.PostForm.Get("name")
	fmt.Println("name", name)
	if oldJob, err = G_JobMgr.DeleteJob(name); err != nil {
		goto ERR
	}
	fmt.Println("oldjob", oldJob)
	if bytes, err = common.BuildResponse(0, "success", oldJob); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	fmt.Println(err.Error())
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

var (
	G_apiServer *ApiServer
)

func InitApiServer() (err error) {
	var (
		mux        *http.ServeMux
		listener   net.Listener
		httpServer *http.Server
	)
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDelete)
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(G_config.ApiPort)); err != nil {
		return
	}
	httpServer = &http.Server{
		ReadTimeout:  time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(G_config.ApiWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}
	G_apiServer = &ApiServer{
		httpServer: httpServer,
	}
	go httpServer.Serve(listener)
	return
}
