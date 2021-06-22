package scheduler

import (
    "context"
    "log"
    "reflect"
    "runtime"
    "strings"
    "sync"
    "time"

    "github.com/bitly/go-simplejson"
    "github.com/heql0668/go-tasks/jobstores"
    "github.com/heql0668/go-tasks/locks"
    "github.com/heql0668/go-tasks/timezone"
    "github.com/trivigy/event"
)

type TaskParams map[string]interface{}
type taskFunc func(params TaskParams) bool

func (f taskFunc) FuncName() string {
    namePath := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
    itemSlice := strings.Split(namePath, ".")
    funcName := itemSlice[len(itemSlice)-1]
    return funcName
}

type Scheduler struct {
    waitSec          int
    ctx              context.Context
    ctxCancel        func()
    handlers         map[string]taskFunc
    taskQueue        chan struct{}
    successTaskCount int
    options          *Options
    mu               sync.Mutex
}

// MaxWorkers: 最大同时开启的协程数量
// MaxAvaiWorkers: 允许最大空闲的携程数量
// TaskMaxRunTime: 每个协程任务运行的最大时长,默认30秒
type Options struct {
    MaxWorkers     int
    MaxAvaiWorkers int
    TaskMaxRunTime int
    Lock           locks.LockIFace
    Store          jobstores.StoreInterface
}

func (sched *Scheduler) processJobs() {
    if sched.ctxCancel != nil {
        defer sched.ctxCancel()
    }
    sched.waitSec = 5
    sched.options.Lock.Lock("processJobs")
    defer sched.options.Lock.Unlock()
    avaiWorkers := sched.options.MaxWorkers - len(sched.taskQueue)
    var jobs []jobstores.Job
    if avaiWorkers >= sched.options.MaxAvaiWorkers {
        log.Printf("当前空闲协程数量为%v, 尝试拉取%v个任务.", avaiWorkers, avaiWorkers)
        jobs = sched.options.Store.GetDueJobs(time.Now().In(timezone.Timezone), avaiWorkers)
    } else {
        return
    }
    var wakeUpTime int
    if len(jobs) == 0 {
        wakeUpTime = sched.options.Store.GetWakeUpTime()
    } else {
        sched.options.Store.LockJobs(jobs, sched.options.TaskMaxRunTime)
        wakeUpTime = sched.options.Store.GetWakeUpTime()
        for _, job := range jobs {
            f := sched.handlers[job.FuncName]
            if f == nil {
                log.Printf("(%v)没有注册", job.FuncName)
                continue
            }
            params, _ := simplejson.NewJson([]byte(job.FuncKwargs))
            sched.taskQueue <- struct{}{}
            go sched.Execute(f, params.MustMap(), job)
        }
    }
    sched.waitSec = wakeUpTime - int(time.Now().In(timezone.Timezone).Unix())
    if sched.waitSec < 0 {
        sched.waitSec = 0
    }
}

func loop(sched *Scheduler) {
    defer func() {
        if err := recover(); err != nil {
            log.Printf("Encountered an error %v", err)
        }
    }()
    sched.taskQueue = make(chan struct{}, sched.options.MaxWorkers)
    e := event.New()
    for {
        if sched.waitSec == 0 {
            sched.ctx, sched.ctxCancel = context.WithTimeout(context.Background(), 0*time.Second)
        } else {
            sched.ctx, sched.ctxCancel = context.WithTimeout(context.Background(), time.Duration(sched.waitSec)*time.Second)
        }
        e.Wait(sched.ctx)
        sched.processJobs()
        nextSchedTime := time.Now().In(timezone.Timezone).Add(time.Second * time.Duration(sched.waitSec)).Format(time.RFC3339)
        log.Printf("调度器下次唤醒时间: %v (%v 秒后), 终止任务数:(%d)\n", nextSchedTime, sched.waitSec, sched.successTaskCount)
    }

}

func (sched *Scheduler) Start(options ...*Options) {
    for {
        loop(sched)
    }
}

func (sched *Scheduler) AddJob(j jobstores.Job) error {
    defer sched.WakeUp()
    sched.options.Store.AddJob(j)
    return nil
}
func (sched *Scheduler) UpdateJob(j jobstores.Job) error {
    defer sched.WakeUp()
    sched.options.Store.UpdateJob(j)
    return nil
}
func (sched *Scheduler) RemoveJob(j jobstores.Job) error {
    defer sched.WakeUp()
    sched.options.Store.RemoveJob(j)
    return nil
}
func (sched *Scheduler) LookupJob(JobID string) jobstores.Job {
    return sched.options.Store.LookupJob(JobID)
}
func (sched *Scheduler) GetAllJobs() []jobstores.Job {
    return sched.options.Store.GetAllJobs()
}
func (sched *Scheduler) RemoveAllJobs() error {
    sched.options.Store.RemoveAllJobs()
    return nil
}

func (sched *Scheduler) RegisterTask(f taskFunc) {
    funcName := f.FuncName()
    if sched.handlers == nil {
        handlers := map[string]taskFunc{funcName: f}
        sched.handlers = handlers
    } else {
        sched.handlers[funcName] = f
    }
}

func (sched *Scheduler) Execute(f taskFunc, params map[string]interface{}, job jobstores.Job) bool {
    defer func() {
        <-sched.taskQueue
        sched.WakeUp()
    }()
    ok := f(params)
    job.SchedTimes++
    now := int(time.Now().In(timezone.Timezone).Unix())
    job.NextRunTime = now + job.SchedTimes*job.IncrStep
    err := sched.options.Store.UpdateJob(job)
    if ok {
        sched.mu.Lock()
        sched.successTaskCount++
        defer sched.mu.Unlock()
        sched.options.Store.RemoveJob(job)
        return true
    }
    if job.MaxSchedTimes > 0 && job.SchedTimes >= job.MaxSchedTimes {
        log.Printf("%v 任务已达最大重试次数(%d)", job.JobID, job.MaxSchedTimes)
    }
    if err == nil {
        log.Printf("任务(%v) 下次被调度时间: %v", job.JobID, time.Unix(int64(job.NextRunTime), 0))
    }
    return false
}

func (sched *Scheduler) WakeUp() {
    if sched.ctxCancel == nil {
        log.Println("Empty context")
        return
    }
    sched.ctxCancel()
}

func (sched *Scheduler) Setup(options *Options) {
    sched.options = options
}

var Sched Scheduler
