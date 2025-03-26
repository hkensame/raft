package raft

//
// support for Raft and kvraft to save persistent
// Raft state (log &c) and k/v server snapshots.
//
// we will use the original persister.go to test your code for grading.
// so, while you can modify this code to help you debug, please
// test with the original before submitting.
//

import (
	"raft/proto/replication"
	"sync"
)

type persister struct {
	mtx         sync.Mutex
	entries     []*replication.Entry
	commitIndex int

	raftstate []byte
	snapshot  []byte
}

func mustNewPersister() *persister {
	return &persister{
		entries: make([]*replication.Entry, 0),
	}
}

func clone(orig []byte) []byte {
	x := make([]byte, len(orig))
	copy(x, orig)
	return x
}

func (ps *persister) Copy() *persister {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	np := mustNewPersister()
	np.raftstate = ps.raftstate
	np.snapshot = ps.snapshot
	return np
}

func (ps *persister) ReadRaftState() []byte {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return clone(ps.raftstate)
}

func (ps *persister) RaftStateSize() int {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return len(ps.raftstate)
}

// Save both Raft state and K/V snapshot as a single atomic action,
// to help avoid them getting out of sync.
func (ps *persister) Save(raftstate []byte, snapshot []byte) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	ps.raftstate = clone(raftstate)
	ps.snapshot = clone(snapshot)
}

func (ps *persister) ReadSnapshot() []byte {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return clone(ps.snapshot)
}

func (ps *persister) SnapshotSize() int {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return len(ps.snapshot)
}
