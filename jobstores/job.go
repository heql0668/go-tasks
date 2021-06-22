package jobstores

import (
    "encoding/json"
    "log"
    "os"
)

type Job struct {
    ID            uint   `json:"id" gorm:"primarykey"`
    JobID         string `json:"job_id" gorm:"uniqueIndex;default:''"`
    FuncName      string `json:"func_name" gorm:"index;default:''"`
    FuncKwargs    string `json:"func_kwargs" gorm:"default:''"`
    SchedTimes    int    `json:"sched_times" gorm:"default:0"`
    IncrStep      int    `json:"incr_step" gorm:"default:3"`
    NextRunTime   int    `json:"next_run_time" gorm:"index;default:0"`
    MaxSchedTimes int    `json:"max_sched_times" gorm:"default:0"`
    CreatedAt     int    `json:"created_at" gorm:"not null"`
    UpdatedAt     int    `json:"updated_at" gorm:"default:0"`
    DeletedAt     int    `json:"deleted_at" gorm:"index;default:0"`
}

func (j Job) String() string {
    b, err := json.Marshal(j)
    if err != nil {
        panic("$j cannot be serialized")
    }
    return string(b)
}

func (Job) TableName() string {
    tbName, ok := os.LookupEnv("JOB_TB_NAME")
    if !ok {
        log.Fatal(`Please set ENV JOB_TB_NAME by os.Setenv("JOB_TB_NAME", "foo")`)
    }
    return tbName
}
