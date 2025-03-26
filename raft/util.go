package raft

import (
	"context"
	"math/rand"
	"time"

	"github.com/hkensame/goken/pkg/log"
)

var electionRange = int64(electionTimeoutMax - electionTimeoutMin)
var replicationRange = int64(replicationTimeoutMax - replicationTimeoutMin)
var sleepRange = int64(sleepTimeMax - sleepTimeMin)

func (r *Raft) getElectionTimeout() time.Duration {
	return electionTimeoutMin + time.Duration(rand.Int63n(electionRange))
}

func (r *Raft) getReplicationTimeout() time.Duration {
	return replicationTimeoutMin + time.Duration(rand.Int63n(replicationRange))
}

func (r *Raft) getSleepTime() time.Duration {
	return sleepTimeMin + time.Duration(rand.Int63n(sleepRange))
}

func (r *Raft) checkTimeout(src time.Time) bool {
	r.tmtx.Lock()
	defer r.tmtx.Unlock()

	return time.Now().After(src)
}

func (r *Raft) resetElectionTicker() {
	r.electionTime = time.Now().Add(r.getElectionTimeout())
}

// 更新一次term,重置身份,得票情况等信息,
func (r *Raft) resetTerm(term int) {
	r.ticketsSource = map[string]int{}
	r.voteFor = ""
	r.currentTerm = int32(term)
	r.totalTickets = 0
}

func (r *Raft) event(ctx context.Context, event string) {
	rowStatus := r.roleFsm.Current()
	if err := r.roleFsm.Event(ctx, event); err != nil {
		if err.Error() == "no transition" {
			return
		}
		log.Errorf("状态机转化失败 触发的event:%s,节点%s从%s状态转变失败 err = %v", event, r.selfInfo.Id, rowStatus, err)
	}
	log.Infof("状态机转化成功 触发的event:%s, 节点%s从%s状态转为%s", event, r.selfInfo.Id, rowStatus, r.roleFsm.Current())
}

// 更新一次term,重置身份,得票情况等信息,函数内会自动加锁
// 注意如果指定的term已经小于节点的term,这个函数不会执行任何信息
// func (r *Raft) resetTermAndLock(term int) {
// 	if ok := r.lock(LockLessTerm(term)); ok {
// 		r.resetTerm(term)
// 	}
// 	r.unlock()
// }

type lockOption func(r *Raft) bool

func LockStatus(status string) lockOption {
	return func(r *Raft) bool {
		return r.roleFsm.Current() == status
	}
}

// 要求等于指定的term
func LockEqualTerm(term int) lockOption {
	return func(r *Raft) bool {
		return r.currentTerm == int32(term)
	}
}

// 要求小于指定的term
func LockLessTerm(term int) lockOption {
	return func(r *Raft) bool {
		return r.currentTerm < int32(term)
	}
}

// 要求大于指定的term
func LockBiggerTerm(term int) lockOption {
	return func(r *Raft) bool {
		return r.currentTerm > int32(term)
	}
}

func LockNotStatus(status string) lockOption {
	return func(r *Raft) bool {
		return r.roleFsm.Current() != status
	}
}

// 这个函数在锁上后会检查自身状态是否符合给定的条件(注意,哪怕不适合也不会解锁,这只是一种提示)
func (r *Raft) slock(opts ...lockOption) bool {
	r.smtx.Lock()
	flag := true
	for _, opt := range opts {
		flag = flag && opt(r)
	}

	return flag
}

// 如果opts条件不满足就解锁
func (r *Raft) smustLock(opts ...lockOption) bool {
	if !r.slock(opts...) {
		r.sunlock()
		return false
	}
	return true
}

func (r *Raft) sunlock() {
	r.smtx.Unlock()
}

func getMajorityNumber(n int) int {
	return n/2 + 1
}
