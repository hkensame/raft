package raft

import (
	"context"
	"math/rand"
	"raft/proto/application"
	"raft/proto/replication"
	"sort"
	"time"

	"github.com/hkensame/goken/pkg/log"
	"google.golang.org/protobuf/proto"
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
	r.tmtx.Lock()
	r.electionTime = time.Now().Add(r.getElectionTimeout())
	r.tmtx.Unlock()
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

func (r *Raft) getMajorityNumber() int {
	return r.raftNodesNumber/2 + 1
}

func (r *Raft) getMajorityIndex() int {
	sortMatchIndex := []int{}
	sortMatchIndex = append(sortMatchIndex, r.getLastIndex())
	for _, v := range r.matchIndex {
		sortMatchIndex = append(sortMatchIndex, v)
	}
	sort.Ints(sortMatchIndex)
	return sortMatchIndex[(len(sortMatchIndex)-1)/2]
}

func (r *Raft) getLastTerm() int {
	return int(r.persister.entries[r.getLastIndex()].Term)
}

func (r *Raft) getLastIndex() int {
	return len(r.persister.entries) - 1
}

func (r *Raft) marshal() []byte {
	a := &application.Metadata{}
	a.PersistedIndex = int32(r.persister.persistedIndex)
	a.Term = r.currentTerm
	b, _ := proto.Marshal(a)
	return b
}

func (r *Raft) unmarshal(b []byte) {
	a := &application.Metadata{}
	proto.Unmarshal(b, a)
	r.persister.persistedIndex = int(a.PersistedIndex)
	r.persister.commitedIndex = r.persister.persistedIndex
	r.persister.pendingIndex = r.persister.persistedIndex
	r.currentTerm = a.Term
}

func (rf *Raft) loadMetadata() {
	b, err := rf.persister.manager.GetCustomData()
	if err != nil {
		panic(err)
	}
	if err := rf.persister.manager.LoadHeaderData(); err != nil {
		panic(err)

	}
	//如果不是新建的raft节点就反序列化
	if b != nil && len(b) != 0 {
		rf.unmarshal(b)
	}

	rf.persister.entries = make([]*replication.Entry, 0, rf.persister.persistedIndex)
	rf.persister.entries = append(rf.persister.entries, &replication.Entry{Term: 0, Command: nil, Index: 0})
	log.Infof("metadata 载入成功 term:%d,commitedIndex:%d", rf.currentTerm, rf.persister.commitedIndex)
}

func (rf *Raft) loadEntries() {
	reader := rf.persister.manager.ReadBlockEntries()
	for {
		data, err := reader.Next()
		if err != nil {
			panic(err)
		}
		if data == nil {
			break
		}
		for _, v := range data {
			ent := &replication.Entry{}
			if err := proto.Unmarshal(v, ent); err != nil {
				panic(err)
			}
			rf.persister.entries = append(rf.persister.entries, ent)
		}
	}
	log.Infof("entries 载入成功 size:%d", len(rf.persister.entries)-1)
}
