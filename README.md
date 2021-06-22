# go-tasks

# 简介
go-task是一个任务重试调度的控制器，可以用于动态新增系统任务，并且在任务执行失败的时候进行重试，重试的时间间隔随着重试次数增加而增大,并支持最大重试次数限制

# 示例

```
package main

import (
    "fmt"
    "log"
    "math/rand"
    "time"

    "github.com/heql0668/go-tasks/jobstores"
    "github.com/heql0668/go-tasks/locks"
    "github.com/heql0668/go-tasks/scheduler"
)

func Demo(params scheduler.TaskParams) bool {
    rand.Seed(time.Now().Local().Unix())
    v := rand.Intn(100)
    mod := v % 2
    sleepSec := rand.Intn(1)
    log.Printf("demo sleep %vs, value: %v, mod: %v", sleepSec, v, mod)
    time.Sleep(time.Duration(sleepSec) * time.Second)
    return mod == 0
}

func AddJobs(sched *scheduler.Scheduler, count int) {
    var job jobstores.Job
    _id := 1
    for i := 0; i < count; i++ {
        jobID := fmt.Sprintf("demo%v", _id)
        job = jobstores.Job{FuncName: "Demo", JobID: jobID, FuncKwargs: `{"name":"foo"}`}
        job.MaxSchedTimes = 3
        _id++
        sched.AddJob(job)
    }
}

func main() {
    var store jobstores.StoreInterface = jobstores.MySQLStore{}
    config := jobstores.Config{
        DBName:   "tasks",
        TBName:   "demo_tasks",
        DBHost:   "127.0.0.1",
        DBPort:   "3306",
        DBUser:   "demo",
        DBPasswd: "123456",
    }
    store.Start(&config)
    locks.Setup(locks.Options{Addr: "redis1.huitouche.io:6380", DB: 10})
    options := scheduler.Options{MaxWorkers: 10, MaxAvaiWorkers: 5, Lock: &locks.RedLock{}, Store: store}
    scheduler.Sched.RegisterTask(Demo)
    scheduler.Sched.Setup(&options)
    AddJobs(&scheduler.Sched, 1000)
    scheduler.Sched.Start()
}
```