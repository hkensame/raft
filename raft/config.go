package raft

import (
	"github.com/hkensame/goken/server/rpcserver"
)

type RaftOption func(*Raft)

func WithPersistFile(filename string) RaftOption {
	return func(rc *Raft) {
		rc.filePath = filename
	}
}

// WithClients设置Raft的客户端映射
func WithClients(clients map[*Instance]*rpcserver.Client) RaftOption {
	return func(r *Raft) {
		r.clients = clients
	}
}

// WithSelfInfo 设置当前实例信息
func WithRaftName(selfInfo *Instance) RaftOption {
	return func(r *Raft) {
		r.selfInfo = selfInfo
	}
}

// WithSelfInfo 设置当前实例信息
func WithRaftId(selfInfo *Instance) RaftOption {
	return func(r *Raft) {
		r.selfInfo = selfInfo
	}
}

// WithRaftNodesNumber 设置 Raft 集群节点数
func WithRaftNodesNumber(nodes int) RaftOption {
	return func(r *Raft) {
		r.raftNodesNumber = nodes
	}
}
