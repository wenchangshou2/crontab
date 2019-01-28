package worker

import (
	"github.com/wenchangshou2/crontab/common"
	"math/rand"
	"os/exec"
	"time"
)

//任务执行器
type Executor struct {
}

var (
	G_executor *Executor
)

func (executor *Executor) ExecuteJob(info *common.JobExecuteInfo) {
	go func() {

		var (
			cmd     *exec.Cmd
			err     error
			output  []byte
			result  *common.JobExecuteResult
			jobLock *JobLock
		)
		result = &common.JobExecuteResult{
			ExecuteInfo: info,
			Output:      make([]byte, 0),
		}

		//初始化锁
		jobLock = G_JobMgr.CreateJobLock(info.Job.Name)

		//记录开始时间
		result.StartTime = time.Now()
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		err = jobLock.TryLock()
		defer jobLock.UnLock()
		if err != nil {
			result.Err = err
			result.EndTime = time.Now()

		} else {
			//上锁成功后
			result.StartTime = time.Now()
			cmd = exec.CommandContext(info.CancelCtx, "bash", "-c", info.Job.Command)
			output, err = cmd.CombinedOutput()
			//记录结束时间
			result.EndTime = time.Now()
			result.Output = output
			result.Err = err
		}
		G_scheduler.PushJobResult(result)
	}()
}

func InitExecutor() (err error) {
	G_executor = &Executor{}
	return
}
