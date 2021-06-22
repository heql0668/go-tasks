package jobstores

import (
    "time"
)

type Config struct {
    DBName   string
    DBUser   string
    DBPasswd string
    DBHost   string
    DBPort   string
    TBName   string
}

type StoreInterface interface {
    Start(*Config)
    LookupJob(ID string) Job
    GetDueJobs(t time.Time, cnt int) []Job
    GetWakeUpTime() (wakeUpTime int)
    AddJob(j Job) Job
    UpdateJob(j Job) error
    LockJobs(jobs []Job, keepTime int) error
    RemoveJob(j Job) error
    GetAllJobs() []Job
    RemoveAllJobs() error
}
