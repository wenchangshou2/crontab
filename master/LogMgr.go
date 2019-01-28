package master

import (
	"context"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/wenchangshou2/crontab/common"
)

type LogMgr struct {
	client        *mongo.Client
	logCollection *mongo.Collection
}

var (
	G_logMgr *LogMgr
)

func InitLogSink() (err error) {
	var (
		client *mongo.Client
	)

	if client, err = mongo.Connect(context.TODO(), G_config.MongodbUri); err != nil {
		return
	}
	G_logMgr = &LogMgr{
		client:        client,
		logCollection: client.Database("cron").Collection("log"),
	}
	return
}

//查看任务日志
func (logMgr *LogMgr) ListLog(name string, skip int, limit int) (logArr []*common.JobLog, err error) {
	var (
		filter     *common.JobLogFilter
		logSort    *common.SortLogByStartTime
		ops        *options.FindOptions
		cursor     mongo.Cursor
		jobLog     *common.JobLog
		skipInt64  int64
		limitInt64 int64
	)
	logArr = make([]*common.JobLog, 0)
	filter = &common.JobLogFilter{JobName: name}
	//按照任务开始时间
	logSort = &common.SortLogByStartTime{SortOrder: -1}
	skipInt64 = int64(skip)
	limitInt64 = int64(limit)
	ops = &options.FindOptions{
		Sort:  logSort,
		Skip:  &skipInt64,
		Limit: &limitInt64,
	}
	if cursor, err = logMgr.logCollection.Find(context.TODO(), filter, ops); err != nil {
		return
	}
	defer cursor.Close(context.TODO())
	for cursor.Next(context.TODO()) {
		jobLog = &common.JobLog{}
		if err = cursor.Decode(jobLog); err != nil {
			continue
		}
		logArr = append(logArr, jobLog)
	}
	return
}
