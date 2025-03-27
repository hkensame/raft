package raft

import (
	"context"
	"raft/proto/replication"
	"sync"

	"github.com/hkensame/goken/pkg/log"
)

type persister struct {
	mtx           sync.Mutex
	entries       []*replication.Entry
	commitedIndex int
	//待提交的index,这个index恒大于等于commitedIndex,但是commitedIndex是持久化的
	pendingIndex int
	persistCond  *sync.Cond

	raftstate []byte
	snapshot  []byte
}

func mustNewPersister() *persister {
	return &persister{
		entries: make([]*replication.Entry, 0),
	}
}

func (r *Raft) applicationTicker(ctx context.Context) {
	for !r.closed {
		select {
		case <-ctx.Done():
			return
		default:
			r.slock()
			r.persister.persistCond.Wait()

			entries := make([]*replication.Entry, 0)
			for i := r.persister.commitedIndex + 1; i <= r.persister.pendingIndex; i++ {
				entries = append(entries, r.persister.entries[i])
			}
			r.sunlock()

			log.Infof("进行一次数据持久化,需要持久的entry个数为%d,预计持久后的commitIndex为%d", len(entries), r.persister.pendingIndex)

			go func() {
				if r.persister.persistence(entries) {
					r.slock()
					r.persister.commitedIndex += len(entries)
					r.sunlock()
				}
			}()
		}
	}
}

// 持久化提交enties,只要把一个entry持久到磁盘就永久固定了一个commitedIndex,不用担心持久化时宕机
// TODO 在这个函数内既需要持久化entries也需要更新commitedIndex
// 现在不需要
func (p *persister) persistence(_ []*replication.Entry) bool {
	return true
}

// NOTICED 这里可能需要返回bool或其他判断成功的可能,后面考虑由哪一层转发
func (r *Raft) AddEntry(c *replication.CommandBody) {
	if r.smustLock(LockStatus(Leader)) {
		ent := &replication.Entry{
			Command: c,
			Term:    r.currentTerm,
			Index:   int32(len(r.persister.entries)),
		}

		r.persister.entries = append(r.persister.entries, ent)
		//NOTICED pending似乎不是在这里加的?
		//r.persister.pendingIndex++
		log.Infof("entry 已经插入,插入的term为:%d,插入的index为:%d", ent.Term, ent.Index)
		r.sunlock()
	}
}

// NOTICED 这里可能需要返回bool或其他判断成功的可能,后面考虑由哪一层转发
func (r *Raft) AddEntries(c []*replication.CommandBody) {
	if r.smustLock(LockStatus(Leader)) {
		for _, v := range c {
			ent := &replication.Entry{
				Command: v,
				Term:    r.currentTerm,
				Index:   int32(len(r.persister.entries)),
			}
			r.persister.entries = append(r.persister.entries, ent)
		}

		log.Infof("entries已经插入2条,插入的term为:%d,插入的起始index为:%d", r.currentTerm, r.getLastIndex()-1)
		r.sunlock()
	}
}

// func clone(orig []byte) []byte {
// 	x := make([]byte, len(orig))
// 	copy(x, orig)
// 	return x
// }

// func (ps *persister) Copy() *persister {
// 	ps.mtx.Lock()
// 	defer ps.mtx.Unlock()
// 	np := mustNewPersister()
// 	np.raftstate = ps.raftstate
// 	np.snapshot = ps.snapshot
// 	return np
// }

// func (ps *persister) ReadRaftState() []byte {
// 	ps.mtx.Lock()
// 	defer ps.mtx.Unlock()
// 	return clone(ps.raftstate)
// }

// func (ps *persister) RaftStateSize() int {
// 	ps.mtx.Lock()
// 	defer ps.mtx.Unlock()
// 	return len(ps.raftstate)
// }

// Save both Raft state and K/V snapshot as a single atomic action,
// to help avoid them getting out of sync.
// func (ps *persister) Save(raftstate []byte, snapshot []byte) {
// 	ps.mtx.Lock()
// 	defer ps.mtx.Unlock()
// 	ps.raftstate = clone(raftstate)
// 	ps.snapshot = clone(snapshot)
// }

// func (ps *persister) ReadSnapshot() []byte {
// 	ps.mtx.Lock()
// 	defer ps.mtx.Unlock()
// 	return clone(ps.snapshot)
// }

// func (ps *persister) SnapshotSize() int {
// 	ps.mtx.Lock()
// 	defer ps.mtx.Unlock()
// 	return len(ps.snapshot)
// }
