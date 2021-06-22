package locks

import (
    "context"
    "log"
    "time"

    goredislib "github.com/go-redis/redis/v8"
    "github.com/go-redsync/redsync/v4"
    "github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

type RedLock struct {
    mu  *redsync.Mutex
    res string
}

func (l *RedLock) Lock(resource string) bool {
    l.res = resource
    mutex := rs.NewMutex(resource)
    l.mu = mutex
    if err := l.mu.Lock(); err != nil {
        return false
    }
    return true
}

func (l *RedLock) Unlock() bool {
    if ok, err := l.mu.Unlock(); !ok || err != nil {
        return false
    }
    return true
}

var rs *redsync.Redsync

// Addr: host:port
// DB: database
type Options struct {
    Addr string
    DB   int
}

func Setup(options Options) {
    client := goredislib.NewClient(&goredislib.Options{Addr: options.Addr, DB: options.DB})
    ctx, cancellFunc := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancellFunc()
    status := client.Ping(ctx)
    _, err := status.Result()
    if err != nil {
        log.Fatal(err)
    }
    pool := goredis.NewPool(client)
    rs = redsync.New(pool)
}
