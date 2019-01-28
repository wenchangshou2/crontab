package master

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/wenchangshou2/crontab/common"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"time"
)

//任务管理器
type JobMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

var (
	G_JobMgr *JobMgr
)

func InitJobMgr() (err error) {
	fmt.Println("initjob")
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
	)
	fmt.Println("config", G_config)
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Microsecond,
	}
	if client, err = clientv3.New(config); err != nil {
		return
	}
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	G_JobMgr = &JobMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}
	return
}
func (jobMgr *JobMgr) SaveJob(job *common.Job) (oldJob *common.Job, err error) {
	var (
		jobKey    string
		jobValue  []byte
		putResp   *clientv3.PutResponse
		oldJobObj common.Job
	)

	jobKey = common.JOB_SAVE_DIR + job.Name
	if jobValue, err = json.Marshal(job); err != nil {
		return
	}
	if putResp, err = jobMgr.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV()); err != nil {
		return
	}
	fmt.Println(putResp.PrevKv)
	if putResp.PrevKv != nil {
		fmt.Println("prev", putResp.PrevKv.Value)
		if err = json.Unmarshal(putResp.PrevKv.Value, &oldJobObj); err != nil {
			err = nil
			return
		}
		oldJob = &oldJobObj
	}
	return
}
func (JobMgr *JobMgr) DeleteJob(name string) (oldJob *common.Job, err error) {
	var (
		jobKey    string
		delResp   *clientv3.DeleteResponse
		oldJobObj common.Job
	)
	jobKey = common.JOB_SAVE_DIR + name
	if delResp, err = JobMgr.kv.Delete(context.TODO(), jobKey, clientv3.WithPrevKV()); err != nil {
		return
	}
	if len(delResp.PrevKvs) != 0 {
		if err = json.Unmarshal(delResp.PrevKvs[0].Value, &oldJobObj); err != nil {
			err = nil
			return
		}
		oldJob = &oldJobObj
	}
	return
}
func (JobMgr *JobMgr) ListJobs() (jobList []*common.Job, err error) {
	var (
		dirKey  string
		getResp *clientv3.GetResponse
		kvPair  *mvccpb.KeyValue
		job     *common.Job
	)
	dirKey = common.JOB_SAVE_DIR
	if getResp, err = JobMgr.kv.Get(context.TODO(), dirKey, clientv3.WithPrefix()); err != nil {
		return
	}
	jobList = make([]*common.Job, 0)
	for _, kvPair = range getResp.Kvs {
		job = &common.Job{}
		if err = json.Unmarshal(kvPair.Value, &job); err != nil {
			err = nil
			continue
		}
		jobList = append(jobList, job)
	}
	return
}
func (JobMgr *JobMgr) KillJob(name string) (err error) {
	var (
		killerKey      string
		leaseGrantResp *clientv3.LeaseGrantResponse
		leaseId        clientv3.LeaseID
	)
	killerKey = common.JOB_KILLER_DIR + name
	if leaseGrantResp, err = JobMgr.lease.Grant(context.TODO(), 1); err != nil {
		return
	}
	leaseId = leaseGrantResp.ID
	if _, err = JobMgr.kv.Put(context.TODO(), killerKey, "", clientv3.WithLease(leaseId)); err != nil {
		return
	}

	return
}
