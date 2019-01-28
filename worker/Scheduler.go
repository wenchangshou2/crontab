package worker

import (
	"fmt"
	"github.com/wenchangshou2/crontab/common"
	"time"
)

type Scheduler struct {
	jobEventChan      chan *common.JobEvent
	jobPlanTable      map[string]*common.JobSchedulePlan
	jobExecutingTable map[string]*common.JobExecuteInfo
	jobResultChan     chan *common.JobExecuteResult
}

var (
	G_scheduler *Scheduler
)

func (scheduler *Scheduler) handleJobEvent(jobEvent *common.JobEvent) {
	var (
		jobSchedulePlan *common.JobSchedulePlan
		jobExistsed     bool
		err             error
		jobExecuteInfo  *common.JobExecuteInfo
		jobExecuting    bool
	)
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE:
		if jobSchedulePlan, err = common.BuildJobSchedulePlan(jobEvent.Job); err != nil {
			return
		}
		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan
	case common.JOB_EVENT_DELETE:
		if jobSchedulePlan, jobExistsed = scheduler.jobPlanTable[jobEvent.Job.Name]; jobExistsed {
			delete(scheduler.jobPlanTable, jobEvent.Job.Name)
		}
	case common.JOB_EVENT_KILL:
		fmt.Println("收到强杀", jobEvent.Job.Name, scheduler.jobExecutingTable)
		if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobEvent.Job.Name]; jobExecuting {
			jobExecuteInfo.CancelFunc() // 触发command杀死shell子进程, 任务得到退出
		}
	}
}

//处理任务结果
func (Scheduler *Scheduler) handleJobResult(result *common.JobExecuteResult) {
	var (
		jobLog *common.JobLog
	)
	delete(Scheduler.jobExecutingTable, result.ExecuteInfo.Job.Name)
	if result.Err != common.ERR_LOCK_ALREADY_REQUIRED {
		jobLog = &common.JobLog{
			JobName:      result.ExecuteInfo.Job.Name,
			Command:      result.ExecuteInfo.Job.Command,
			Output:       string(result.Output),
			PlanTime:     result.ExecuteInfo.PlanTime.UnixNano() / 1000 / 1000,
			ScheduleTime: result.ExecuteInfo.RealTime.UnixNano() / 1000 / 1000,
			StartTime:    result.StartTime.UnixNano() / 1000 / 1000,
			EndTime:      result.EndTime.UnixNano() / 1000 / 1000,
		}
		if result.Err != nil {
			jobLog.Err = result.Err.Error()
		} else {
			jobLog.Err = ""
		}
		G_logSink.Append(jobLog)
	}

}

//尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobPlan *common.JobSchedulePlan) {
	// 调度 和 执行 是2件事情
	var (
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuting   bool
	)

	// 执行的任务可能运行很久, 1分钟会调度60次，但是只能执行1次, 防止并发！

	// 如果任务正在执行，跳过本次调度
	if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobPlan.Job.Name]; jobExecuting {
		// fmt.Println("尚未退出,跳过执行:", jobPlan.Job.Name)
		return
	}

	// 构建执行状态信息
	jobExecuteInfo = common.BuildJobExecuteInfo(jobPlan)

	// 保存执行状态
	scheduler.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo

	// 执行任务
	fmt.Println("执行任务:", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	G_executor.ExecuteJob(jobExecuteInfo)

}
func (scheduler *Scheduler) TrySchedule() (scheduleAfter time.Duration) {
	var (
		jobPlan  *common.JobSchedulePlan
		now      time.Time
		nearTime *time.Time
	)
	if len(scheduler.jobPlanTable) == 0 {
		scheduleAfter = 1 * time.Second
		return
	}
	now = time.Now()
	for _, jobPlan = range scheduler.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) { //是否轮到它执行
			fmt.Println("执行任务：", jobPlan.Job.Name)
			scheduler.TryStartJob(jobPlan)
			jobPlan.NextTime = jobPlan.Expr.Next(now)
		}
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}
	scheduleAfter = (*nearTime).Sub(now)
	return scheduleAfter
}
func (scheduler *Scheduler) scheduleLoop() {
	var (
		jobEvent      *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobResult     *common.JobExecuteResult
	)
	scheduleAfter = scheduler.TrySchedule()
	//调度延迟定时器
	scheduleTimer = time.NewTimer(scheduleAfter)
	for {
		select {
		case jobEvent = <-scheduler.jobEventChan:
			scheduler.handleJobEvent(jobEvent)
		case <-scheduleTimer.C:
		case jobResult = <-scheduler.jobResultChan: //监听任务执行结果
			scheduler.handleJobResult(jobResult)
		}
		scheduleAfter = scheduler.TrySchedule()
		scheduleTimer.Reset(scheduleAfter)
	}
}
func (scheduler *Scheduler) PushJobEvent(jobEvent *common.JobEvent) {
	scheduler.jobEventChan <- jobEvent

}
func InitScheduler() (err error) {
	G_scheduler = &Scheduler{
		jobEventChan:      make(chan *common.JobEvent, 1000),
		jobPlanTable:      make(map[string]*common.JobSchedulePlan),
		jobExecutingTable: make(map[string]*common.JobExecuteInfo),
		jobResultChan:     make(chan *common.JobExecuteResult, 1000),
	}
	go G_scheduler.scheduleLoop()
	return
}
func (scheduler *Scheduler) PushJobResult(jobResult *common.JobExecuteResult) {
	scheduler.jobResultChan <- jobResult
}
