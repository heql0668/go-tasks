package jobstores

import (
    "fmt"
    "log"
    "os"
    "time"

    "github.com/heql0668/go-tasks/timezone"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

type MySQLStore struct{}

var db *gorm.DB

func (s MySQLStore) Start(config *Config) {
    var err error
    os.Setenv("JOB_TB_NAME", config.TBName)
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", config.DBUser, config.DBPasswd, config.DBHost, config.DBPort, config.DBName)
    db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        panic("Cannot connect to database.")
    }
    sqlDB, err := db.DB()
    if err != nil {
        panic(err)
    }
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(30)
    db.AutoMigrate(&Job{})

}

func (s MySQLStore) GetDueJobs(t time.Time, cnt int) []Job {
    var jobs []Job
    db.Where(
        "deleted_at = 0 AND next_run_time <= ? AND ((sched_times < max_sched_times) or (max_sched_times=0))",
        int(time.Now().In(timezone.Timezone).Unix())).Order("next_run_time asc").Limit(cnt).Find(&jobs)
    return jobs
}

func (s MySQLStore) GetWakeUpTime() (wakeUpTime int) {
    var job Job
    result := db.Where("deleted_at = 0 AND ((sched_times < max_sched_times) or (max_sched_times=0))").Order("next_run_time asc").Select("next_run_time").First(&job)
    if result.Error == gorm.ErrRecordNotFound {
        log.Println("No record found.")
    }
    if result.Error != nil {
        return int(time.Now().In(timezone.Timezone).Unix()) + 5
    }
    return job.NextRunTime
}

func (s MySQLStore) AddJob(j Job) Job {
    db.Create(&j)
    log.Printf("new job: %v", j)
    return j
}
func (s MySQLStore) UpdateJob(j Job) error {
    db.Save(&j)
    return nil
}
func (s MySQLStore) RemoveJob(j Job) error {
    j.SchedTimes++
    j.DeletedAt = int(time.Now().In(timezone.Timezone).Unix())
    db.Save(j)
    return nil
}
func (s MySQLStore) LookupJob(ID string) Job {
    var job Job
    result := db.Where("job_id = ?", ID).First(job)
    if result.Error == gorm.ErrRecordNotFound {
        log.Println("No record found")
    }
    return job
}
func (s MySQLStore) GetAllJobs() []Job {
    var jobs []Job
    db.Where("deleted_at = 0").Find(&jobs)
    return jobs
}
func (s MySQLStore) RemoveAllJobs() error {
    db.Model(Job{}).Where("deleted_at = ?", 0).Updates(Job{DeletedAt: int(time.Now().In(timezone.Timezone).Unix())})
    return nil
}

// 当前服务实例拉取任务下来上锁避免其他实例重复获取相同的任务
// keepTime: 每个协程任务最大的运行时长
func (s MySQLStore) LockJobs(jobs []Job, keepTime int) error {
    if keepTime <= 1 {
        keepTime = 30
    }
    var recordIDs []int
    for _, v := range jobs {
        recordIDs = append(recordIDs, int(v.ID))
    }
    nextRunTime := int(time.Now().In(timezone.Timezone).Unix()) + keepTime
    db.Model(&Job{}).Where("id IN ?", recordIDs).Updates(Job{NextRunTime: nextRunTime})
    return nil
}
