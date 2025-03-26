package raft

import (
	"context"
	"raft/proto/replication"
	"time"

	"github.com/hkensame/goken/pkg/log"
)

func (r *Raft) replicationTicker(ctx context.Context) {
	for !r.closed {
		if !r.smustLock(LockStatus(Leader)) {
			log.Infof("节点%s状态已经不再是leader,退出日志同步函数", r.selfInfo.Id)
			return
		}

		// select {
		// //存在客户端请求
		// case ent := <-r.httpChan:
		// 	r.lastLogginIndex++
		// 	r.lastLogginTerm = int(r.currentTerm)
		// default:
		// }

		r.sunlock()

		select {
		case <-ctx.Done():
			return
		default:
			log.Infof("leader节点%s的日志同步状态为: term:%d", r.selfInfo.Id, r.currentTerm)
			log.Infof("leader节点%s发送一次replicate", r.selfInfo.Id)
			//go r.requestReplicated(ctx)
			go r.requestAppendEntries(ctx)
			time.Sleep(r.getReplicationTimeout())
		}
	}
}

func (r *Raft) requestReplicated(ctx context.Context) error {
	req := &replication.ReplicatedReq{}
	if r.slock(LockStatus(Leader)) {
		req.LeaderId = r.selfInfo.Id
		req.LeaderTerm = r.currentTerm

		r.sunlock()
	} else {
		log.Infof("节点%s状态已经不再是leader,退出日志同步函数", r.selfInfo.Id)
		r.sunlock()
		return ErrInvalidStatus
	}

	for k, v := range r.clients {
		if !r.slock(LockStatus(Leader)) {
			log.Infof("节点%s状态已经不再是leader,退出日志同步函数", r.selfInfo.Id)
			r.sunlock()
			return ErrInvalidStatus
		}

		cli, err := v.Dial()
		if err != nil {
			r.sunlock()
			log.Errorf("raft对端不可达,对端信息为:%s, err = %v", v.Endpoint.String(), err)
			continue
		}

		res, err := replication.NewReplicationClient(cli).Replicated(r.ctx, req)
		if err != nil {
			r.sunlock()
			log.Errorf("raft调用对端ReceiveVote函数失败,对端信息为:%s, err = %v", v.Endpoint.String(), err)
			continue
		}

		if res.Term > r.currentTerm {
			log.Infof("leader节点%s从日志同步res中发现%s节点是比自己大的term", r.selfInfo.Id, k.Id)
			r.resetTerm(int(res.Term))
			r.tmtx.Lock()
			r.resetElectionTicker()
			r.tmtx.Unlock()
			r.event(ctx, EventLessTerm)
			r.sunlock()
			return ErrInvalidStatus
		}
		r.sunlock()
	}
	return nil
}

func (r *Raft) requestAppendEntries(ctx context.Context) error {
	req := &replication.AppendEntriesReq{}
	if r.slock(LockStatus(Leader)) {
		req.LeaderId = r.selfInfo.Id
		req.LeaderTerm = r.currentTerm
		req.LeaderCommitIndex = int32(r.presister.commitIndex)
		r.sunlock()
	} else {
		log.Infof("节点%s状态已经不再是leader,退出日志同步函数", r.selfInfo.Id)
		r.sunlock()
		return ErrInvalidStatus
	}

	for k, v := range r.clients {
		if !r.slock(LockStatus(Leader)) {
			log.Infof("节点%s状态已经不再是leader,退出日志同步函数", r.selfInfo.Id)
			r.sunlock()
			return ErrInvalidStatus
		}

		cli, err := v.Dial()
		if err != nil {
			r.sunlock()
			log.Errorf("raft对端不可达,对端信息为:%s, err = %v", v.Endpoint.String(), err)
			continue
		}

		//这里靠着raft初始化时插入了一条空的entry保证了nextIndex-1恒大于-1且entries切片永远不会越界
		req.PrevIndex = int32(r.nextIndex[k.Id]) - 1
		req.PrevTerm = int32(r.presister.entries[req.PrevIndex].Term)
		req.Entries = r.presister.entries[req.PrevIndex+1:]

		res, err := replication.NewReplicationClient(cli).AppendEntries(r.ctx, req)
		if err != nil {
			r.sunlock()
			log.Errorf("raft调用对端ReceiveVote函数失败,对端信息为:%s, err = %v", v.Endpoint.String(), err)
			continue
		}

		if res.Term > r.currentTerm {
			log.Infof("leader节点%s从日志同步res中发现%s节点是比自己大的term", r.selfInfo.Id, k.Id)
			r.resetTerm(int(res.Term))
			r.tmtx.Lock()
			r.resetElectionTicker()
			r.tmtx.Unlock()
			r.event(ctx, EventLessTerm)
			r.sunlock()
			return ErrInvalidStatus
		}
		//日志没有对齐
		if !res.Align {
			log.Infof("leader节点%s从日志同步res中发现%s节点是不匹配传送的日志的", r.selfInfo.Id, k.Id)
			log.Infof("leader提供了term为%d,index为%d的不合适的日志", req.PrevTerm, req.PrevIndex)

			//寻找前一条index,并且保证找到的entry不是已经被提交的
			if r.nextIndex[k.Id] > 1 {
				r.nextIndex[k.Id]--
			}
		} else {
			//
			r.matchIndex[k.Id] = int(req.PrevIndex) + len(req.Entries)
			r.nextIndex[k.Id] = r.matchIndex[k.Id] + 1
		}
		r.sunlock()
	}
	return nil
}

func (r *Raft) Replicated(ctx context.Context, in *replication.ReplicatedReq) (*replication.ReplicatedRes, error) {
	res := &replication.ReplicatedRes{
		Health: true,
	}

	r.slock()
	defer r.sunlock()
	res.Term = r.currentTerm

	//收到leader的call且leaderTerm大于等于自己则结束一切选举行为
	//如果老leader收到了则还需要开启ticker
	if in.LeaderTerm >= r.currentTerm {
		log.Infof("节点%s从日志复制请求req中发现比自己大的leader term,归属leader", r.selfInfo.Id)
		r.currentTerm = in.LeaderTerm
		r.resetTerm(int(in.LeaderTerm))
		r.tmtx.Lock()
		r.resetElectionTicker()
		r.tmtx.Unlock()
		r.event(ctx, EventLeaderCall)
		r.leaderId = in.LeaderId
		r.hasLeader = true

		return res, nil
	} else {
		log.Infof("节点%s从日志复制请求req中发现比自己小的leader term,忽视", r.selfInfo.Id)
		//leader的任期小于自己则返回象征不可用的信息
		res.Health = false
		return res, nil
	}

}

func (r *Raft) AppendEntries(ctx context.Context, in *replication.AppendEntriesReq) (*replication.AppendEntriesRes, error) {
	res := &replication.AppendEntriesRes{
		Health: true,
	}

	r.slock()
	defer r.sunlock()
	res.Term = r.currentTerm

	//收到leader的call且leaderTerm大于等于自己则结束一切选举行为
	//如果老leader收到了则还需要开启ticker
	if in.LeaderTerm >= r.currentTerm {
		log.Infof("节点%s从日志复制请求req中发现比自己大的leader term,归属leader", r.selfInfo.Id)
		r.currentTerm = in.LeaderTerm
		r.resetTerm(int(in.LeaderTerm))
		r.tmtx.Lock()
		r.resetElectionTicker()
		r.tmtx.Unlock()
		r.event(ctx, EventLeaderCall)
		r.leaderId = in.LeaderId
		r.hasLeader = true

		//尝试找到来自leader指定的prevIndex日志序列
		if int(in.PrevIndex) <= len(r.presister.entries) {
			if r.presister.entries[in.PrevIndex].Term == in.PrevTerm {
				res.Align = true
				r.presister.entries = append(r.presister.entries[:in.PrevIndex+1], in.Entries...)
				r.lastLogginIndex = len(r.presister.entries)
				r.lastLogginTerm = int(r.presister.entries[len(r.presister.entries)-1].Term)
			}
		}
		return res, nil
	} else {
		log.Infof("节点%s从日志复制请求req中发现比自己小的leader term,忽视", r.selfInfo.Id)
		//leader的任期小于自己则返回象征不可用的信息
		res.Health = false
		return res, nil
	}

}
