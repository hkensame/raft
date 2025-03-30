package raft

import (
	"github.com/hkensame/goken/server/rpcserver"
)

type RaftOption func(*Raft)
type ConfOption func(*RaftConf)

type RaftConf struct {
	persistFile string
}

func MustNewDefaultRaftConf() *RaftConf {
	return &RaftConf{}
}

func MustNewRaftConf(opts ...ConfOption) *RaftConf {
	r := MustNewDefaultRaftConf()
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func WithPersistFile(filename string) ConfOption {
	return func(rc *RaftConf) {
		rc.persistFile = filename
	}
}

func WithConf(conf *RaftConf) RaftOption {
	return func(r *Raft) {
		r.conf = conf
	}
}

// WithClients设置Raft的客户端映射
func WithClients(clients map[Instance]*rpcserver.Client) RaftOption {
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
