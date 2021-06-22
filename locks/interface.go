package locks

type LockIFace interface {
    Lock(resource string) bool
    Unlock() bool
}
