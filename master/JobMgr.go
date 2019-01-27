package master

import (
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"time"
)
//任务管理器
type JobMgr struct{
	client *clientv3.Client
	kv clientv3.KV
	lease clientv3.Lease
}
var (
	G_JobMgr *JobMgr
)
func InitJobMgr()(err error){
	fmt.Println("initjob")
	var (
		config clientv3.Config
		client *clientv3.Client
		kv clientv3.KV
		lease clientv3.Lease
	)
	fmt.Println("config",G_config.EtcdEndpoints)
	config=clientv3.Config{
		Endpoints:G_config.EtcdEndpoints ,
		DialTimeout:time.Duration(G_config.EtcdDialTimeout)*time.Microsecond,
	}
	if client,err=clientv3.New(config);err!=nil{
		return
	}
	kv=clientv3.NewKV(client)
	lease=clientv3.NewLease(client)
	G_JobMgr=&JobMgr{
		client:client,
		kv:kv,
		lease:lease,
	}
	return
}