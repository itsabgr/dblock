package dblock

import (
	"context"
	"database/sql"
	"dblock/cancelable"
	"dblock/ent"
	"dblock/ent/dlock"
	"errors"
	"github.com/google/uuid"
	"log/slog"
	"sync"
	"time"
)

func IsLockingFailed(err error) bool {
	return ent.IsConstraintError(err)
}

type Locker struct {
	db           *ent.Client
	Logger       *slog.Logger
	leaseSeconds int64
}

func (l *Locker) log(level slog.Level, msg string, args ...any) {
	if l.Logger == nil {
		return
	}
	l.Logger.Log(context.Background(), level, msg, args...)
}
func (l *Locker) timeout() time.Duration {
	return max((time.Duration(l.leaseSeconds)*time.Second)/2, time.Second*2)
}

const LeaseSeconds = 5

func New(db *ent.Client, leaseSeconds int64) *Locker {
	if leaseSeconds <= 2 {
		panic(errors.New("too less leaseSeconds"))
	}
	if leaseSeconds > 120 {
		panic(errors.New("too big leaseSeconds"))
	}
	return &Locker{
		db:           db,
		leaseSeconds: leaseSeconds,
	}
}

type Lock struct {
	updateLock       sync.Mutex
	parent           *Locker
	id               string
	holder           uuid.UUID
	heartbeatSeconds int64
	heartbeater      *cancelable.Cancelable //volatile
}

func (l *Lock) deadlineSeconds(now int64) int64 {
	if now <= 0 {
		now = time.Now().UTC().Unix()
	}
	return now + l.heartbeatSeconds + l.parent.leaseSeconds
}
func (l *Lock) update(ctx context.Context) error {
	l.updateLock.Lock()
	defer l.updateLock.Unlock()
	if l.heartbeater.Canceled() {
		return errors.New("canceled")
	}
	timeout, cancel := context.WithTimeout(ctx, l.parent.timeout())
	defer cancel()
	deadline := l.deadlineSeconds(0)
	affected, err := l.parent.db.DLock.Update().Where(dlock.And(dlock.ID(l.id), dlock.HolderEQ(l.holder))).SetDeadline(deadline).Save(timeout)
	if err == nil && affected != 1 {
		err = errors.New("affected records count is not one")
	}
	if err == nil {
		l.parent.log(slog.LevelInfo, "lock updated", "id", l.id, "holder", l.holder, "deadline", deadline)
	} else {
		l.parent.log(slog.LevelWarn, "lock update failed", "id", l.id, "holder", l.holder, "deadline", deadline, "err", err)
	}
	return err
}
func (l *Lock) Name() string {
	return l.id
}

func (l *Lock) delete(ctx context.Context) {
	l.updateLock.Lock()
	defer l.updateLock.Unlock()
	timeout, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	err := l.parent.db.DLock.DeleteOneID(l.id).Where(dlock.HolderEQ(l.holder)).Exec(timeout)
	if err == nil {
		l.parent.log(slog.LevelInfo, "lock deleted", "id", l.id, "holder", l.holder)
	} else {
		l.parent.log(slog.LevelWarn, "lock delete failed", "id", l.id, "holder", l.holder, "err", err)
	}
}

func (l *Lock) heartbeat(canc *cancelable.Cancelable) {
	l.parent.log(slog.LevelInfo, "heartbeater started", "id", l.id, "holder", l.holder)
	defer l.parent.log(slog.LevelInfo, "heartbeater canceled", "id", l.id, "holder", l.holder)
	defer canc.Cancel(errors.New("heartbeat failed"))
	var err error
	sleepDuration := time.Duration(l.heartbeatSeconds) * time.Second
	for canc.Ctx().Err() == nil {
		l.parent.log(slog.LevelDebug, "heartbeater sleep", "id", l.id, "holder", l.holder, "duration", sleepDuration)
		if err = cancelable.SleepCtx(canc.Ctx(), sleepDuration); err != nil {
			l.parent.log(slog.LevelInfo, "heartbeater ctx closed", "id", l.id, "holder", l.holder, "err", err)
			return
		}
		if err = l.update(canc.Ctx()); err != nil {
			l.parent.log(slog.LevelWarn, "heartbeater update failed", "id", l.id, "holder", l.holder, "err", err)
			canc.Cancel(err)
			return
		}
		l.parent.log(slog.LevelInfo, "heartbeat", "id", l.id, "holder", l.holder)
	}
}

func (l *Lock) Ctx() context.Context {
	return l.heartbeater.Ctx()
}

func (l *Lock) Active() bool {
	return !l.heartbeater.Canceled()
}

func (l *Lock) Release() bool {
	if l.heartbeater.Cancel(errors.New("released")) {
		l.parent.log(slog.LevelInfo, "lock released", l.id, l.holder)
		l.delete(context.Background())
		return true
	}
	return false
}

func (l *Locker) TryAcquire(pctx context.Context, id string, heartbeatSeconds int64) (*Lock, error) {
	if heartbeatSeconds <= 0 {
		panic(errors.New("non-positive heartbeatSeconds"))
	}
	if heartbeatSeconds > 31556952 {
		panic(errors.New("too big heartbeatSeconds"))
	}
	lock := &Lock{
		parent:           l,
		id:               id,
		holder:           uuid.New(),
		heartbeatSeconds: heartbeatSeconds,
	}

	timeout, cancel := context.WithTimeout(pctx, l.timeout())
	defer cancel()
	tx, err := l.db.BeginTx(timeout, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	})
	if err != nil {
		l.log(slog.LevelDebug, "failed to begin lock tx", "id", id, err)
		return nil, err
	}

	defer tx.Rollback()

	now := time.Now().UTC().Unix()

	deadline := lock.deadlineSeconds(now)

	if err := tx.DLock.DeleteOneID(id).Where(dlock.DeadlineLT(now)).Exec(timeout); err != nil {
		if !ent.IsNotFound(err) {
			l.log(slog.LevelDebug, "failed to delete old lock", "id", id, err)
			return nil, err
		}
	}

	if err = tx.DLock.Create().SetID(id).SetHolder(lock.holder).SetDeadline(deadline).Exec(timeout); err != nil {
		l.log(slog.LevelDebug, "failed to create lock", "id", id, err)
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		l.log(slog.LevelDebug, "failed to commit lock tx", "id", id, err)
		return nil, err
	}
	l.log(slog.LevelInfo, "new lock", "id", lock.id, "holder", lock.holder, "hrs", lock.heartbeatSeconds)
	lock.heartbeater = cancelable.New(pctx, lock.heartbeat)
	return lock, nil

}
