package worker

import (
	"context"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/wenchangshou2/crontab/common"
	"time"
)

//mongodb存储日志
type LogSink struct {
	client         *mongo.Client
	logCollection  *mongo.Collection
	logChan        chan *common.JobLog
	autoCommitChan chan *common.LogBatch
}

var (
	G_logSink *LogSink
)

func (LogSink *LogSink) saveLogs(batch *common.LogBatch) {
	LogSink.logCollection.InsertMany(context.TODO(), batch.Logs)
}
func (logSink *LogSink) WriteLoop() {
	var (
		log          *common.JobLog
		logBatch     *common.LogBatch
		commitTimer  *time.Timer
		timeoutBatch *common.LogBatch
	)
	for {
		select {
		case log = <-logSink.logChan:
			if logBatch == nil {
				logBatch = &common.LogBatch{}
				commitTimer = time.AfterFunc(time.Duration(G_config.JobLogCommitTimeout)*time.Millisecond,
					func(batch *common.LogBatch) func() {
						return func() {
							logSink.autoCommitChan <- batch
						}
					}(logBatch))
			}
			//把新的日志追加
			logBatch.Logs = append(logBatch.Logs, log)
			if len(logBatch.Logs) >= G_config.JobLogBatchSize {
				logSink.saveLogs(logBatch)
				logBatch = nil
				commitTimer.Stop()
			}
		case timeoutBatch = <-logSink.autoCommitChan: //过期的批次
			if timeoutBatch != logBatch {
				continue
			}
			//把过期的日志写入到mongo
			logSink.saveLogs(timeoutBatch)
			logBatch = nil
		}
	}
}
func InitLogSink() (err error) {
	var (
		client *mongo.Client
	)

	if client, err = mongo.Connect(context.TODO(), G_config.MongodbUri); err != nil {
		return
	}
	G_logSink = &LogSink{
		client:         client,
		logCollection:  client.Database("cron").Collection("log"),
		logChan:        make(chan *common.JobLog, 1000),
		autoCommitChan: make(chan *common.LogBatch, 1000),
	}
	go G_logSink.WriteLoop()
	return
}
func (logSink *LogSink) Append(jobLog *common.JobLog) {
	select {
	case logSink.logChan <- jobLog:
	default:

	}
}
