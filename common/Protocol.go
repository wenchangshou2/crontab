package common

import (
	"context"
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"strings"
	"time"
)

type Job struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	CronExpr string `json:"cronExpr"`
}
type JobSchedulePlan struct {
	Job      *Job
	Expr     *cronexpr.Expression
	NextTime time.Time
}
type JobExecuteInfo struct {
	Job        *Job
	PlanTime   time.Time
	RealTime   time.Time
	CancelCtx  context.Context
	CancelFunc context.CancelFunc
}
type Response struct {
	Errno int         `json:"errno"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}
type JobEvent struct {
	EventType int
	Job       *Job
}
type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo
	Err         error     //异常
	Output      []byte    //输出
	StartTime   time.Time //启动时间
	EndTime     time.Time //结束 时间
}
type JobLog struct {
	JobName      string `bson:"jobName"`
	Command      string `bson:"command"`
	Err          string `bson:"err"`
	Output       string `bson:"output"`
	PlanTime     int64  `bson:"planTIme"`     //计划开始时间
	ScheduleTime int64  `bson:"scheduleTime"` //实际调试调试时间
	StartTime    int64  `bson:"startTime"`    //任务执行开始时间
	EndTime      int64  `bson:"endTime"`      //任务执行结束时间
}
type LogBatch struct {
	Logs []interface{} //多条日志
}
type JobLogFilter struct {
	JobName string `bson:"jobName"`
}

//任务日志排序
type SortLogByStartTime struct {
	SortOrder int `bson:"startTime"` //按startTime:-1,
}

func BuildResponse(errno int, msg string, data interface{}) (resp []byte, err error) {
	var (
		response Response
	)
	response.Errno = errno
	response.Msg = msg
	response.Data = data
	resp, err = json.Marshal(response)
	return
}
func UnpackJob(value []byte) (ret *Job, err error) {
	var (
		job *Job
	)
	job = &Job{}
	if err = json.Unmarshal(value, job); err != nil {
		return
	}
	ret = job
	return
}

//从etcd的key提取任务名
func ExtractJobName(jobKey string) string {
	return strings.TrimPrefix(jobKey, JOB_SAVE_DIR)
}
func ExtractKillerName(killerKey string) string {
	return strings.TrimPrefix(killerKey, JOB_KILLER_DIR)
}
func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {

	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}
func BuildJobSchedulePlan(job *Job) (jobSchedulePlan *JobSchedulePlan, err error) {
	var (
		expr *cronexpr.Expression
	)
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return
	}
	jobSchedulePlan = &JobSchedulePlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}
	return
}
func BuildJobExecuteInfo(jobSchedulePaln *JobSchedulePlan) (jobExecuteInfo *JobExecuteInfo) {
	jobExecuteInfo = &JobExecuteInfo{
		Job:      jobSchedulePaln.Job,
		PlanTime: jobSchedulePaln.NextTime,
		RealTime: time.Now(),
	}
	jobExecuteInfo.CancelCtx, jobExecuteInfo.CancelFunc = context.WithCancel(context.TODO())
	return
}
