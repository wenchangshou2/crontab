package worker

import (
	"context"
	"fmt"
	"github.com/wenchangshou2/crontab/common"
	"go.etcd.io/etcd/clientv3"
)

type JobLock struct {
	kv         clientv3.KV
	lease      clientv3.Lease
	jobName    string
	cancelFunc context.CancelFunc //用于终止续租
	leaseId    clientv3.LeaseID
	isLocked   bool
}

//初始化一把锁
func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) (jobLock *JobLock) {
	jobLock = &JobLock{
		kv:      kv,
		lease:   lease,
		jobName: jobName,
	}
	return
}
func (jobLock *JobLock) TryLock() (err error) {
	var (
		leaseGrantResp *clientv3.LeaseGrantResponse
		cancelCtx      context.Context
		cancelFunc     context.CancelFunc
		leaseId        clientv3.LeaseID
		keepRespChan   <-chan *clientv3.LeaseKeepAliveResponse
		txn            clientv3.Txn
		lockKey        string
		txnResp        *clientv3.TxnResponse
	)
	//1.创建租约(5秒）
	if leaseGrantResp, err = jobLock.lease.Grant(context.TODO(), 5); err != nil {
		return
	}
	cancelCtx, cancelFunc = context.WithCancel(context.TODO())
	leaseId = leaseGrantResp.ID
	if keepRespChan, err = jobLock.lease.KeepAlive(cancelCtx, leaseId); err != nil {
		goto FAIL
	}
	go func() {
		var (
			keepResp *clientv3.LeaseKeepAliveResponse
		)
		for {
			select {
			case keepResp = <-keepRespChan: //自动续租的应答
				if keepResp == nil {
					goto END
				}
			}
		}
	END:
	}()
	//2.自动续租
	//3创建事务txn
	txn = jobLock.kv.Txn(context.TODO())
	//5事务抢锁

	lockKey = common.JOB_LOCK_DIR + jobLock.jobName
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseId))).
		Else(clientv3.OpGet(lockKey))
	if txnResp, err = txn.Commit(); err != nil {
		goto FAIL
	}
	//6成功返回，失败释放租约
	if !txnResp.Succeeded {
		fmt.Println("锁被占用")
		err = common.ERR_LOCK_ALREADY_REQUIRED
		goto FAIL
	}
	jobLock.leaseId = leaseId
	jobLock.cancelFunc = cancelFunc
	jobLock.isLocked = true
	return
FAIL:
	cancelFunc()
	jobLock.lease.Revoke(context.TODO(), leaseId) //释放租约
	return
}
func (JobLock *JobLock) UnLock() {
	if JobLock.isLocked {
		JobLock.cancelFunc()
		JobLock.lease.Revoke(context.TODO(), JobLock.leaseId)
	}

}
