package worker

import (
	"context"
	"fmt"
	"github.com/wenchangshou2/crontab/common"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"time"
)

//任务管理器
type JobMgr struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
}

var (
	G_JobMgr *JobMgr
)

func InitJobMgr() (err error) {
	fmt.Println("initjob")
	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		watcher clientv3.Watcher
	)
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Microsecond,
	}
	if client, err = clientv3.New(config); err != nil {
		return
	}
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	watcher = clientv3.NewWatcher(client)
	G_JobMgr = &JobMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}
	G_JobMgr.watchJobs()
	G_JobMgr.watchKiller()
	return
}

//创建任务执行锁
func (jobMgr *JobMgr) CreateJobLock(jobName string) (jobLock *JobLock) {
	jobLock = InitJobLock(jobName, jobMgr.kv, jobMgr.lease)
	return
}
func (jobMgr *JobMgr) watchKiller() {
	var (
		watchChan  clientv3.WatchChan
		watchResp  clientv3.WatchResponse
		watchEvent *clientv3.Event
		jobEvent   *common.JobEvent
		jobName    string
		job        *common.Job
	)
	// 监听/cron/killer目录
	go func() { // 监听协程
		// 监听/cron/killer/目录的变化
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_KILLER_DIR, clientv3.WithPrefix())
		// 处理监听事件
		for watchResp = range watchChan {
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: // 杀死任务事件
					jobName = common.ExtractKillerName(string(watchEvent.Kv.Key))
					job = &common.Job{Name: jobName}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_KILL, job)
					// 事件推给scheduler
					G_scheduler.PushJobEvent(jobEvent)
				case mvccpb.DELETE: // killer标记过期, 被自动删除
				}
			}
		}
	}()

}
func (jobMgr *JobMgr) watchJobs() (err error) {
	var (
		getResp            *clientv3.GetResponse
		kvpair             *mvccpb.KeyValue
		job                *common.Job
		watchStartRevision int64
		watchChan          clientv3.WatchChan
		watchResp          clientv3.WatchResponse
		watchEvent         *clientv3.Event
		jobName            string
		jobEvent           *common.JobEvent
	)
	if getResp, err = jobMgr.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix()); err != nil {
		return
	}
	fmt.Println(getResp.Kvs)
	for _, kvpair = range getResp.Kvs {
		fmt.Println("value", kvpair.Value)
		if job, err = common.UnpackJob(kvpair.Value); err == nil {
			fmt.Println("job", job.Command, job.Name, job.CronExpr)
			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
			fmt.Println("jobevent", jobEvent)
			G_scheduler.PushJobEvent(jobEvent)
		}
	}
	go func() { //监听协程
		watchStartRevision = getResp.Header.Revision + 1
		fmt.Println(watchStartRevision)
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithRev(watchStartRevision), clientv3.WithPrefix())
		for watchResp = range watchChan {
			for _, watchEvent = range watchResp.Events {

				switch watchEvent.Type {
				case mvccpb.PUT:
					if job, err = common.UnpackJob(watchEvent.Kv.Value); err != nil {
						continue
					}
					//TODO：反序列化job，推送一个更新给scheduler
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
				case mvccpb.DELETE:
					jobName = common.ExtractJobName(string(watchEvent.Kv.Key))
					job = &common.Job{Name: jobName}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE, job)

					//TODO: 推一个删除事件给scheduler
				}
				G_scheduler.PushJobEvent(jobEvent)
			}
		}
	}()

	return
}
