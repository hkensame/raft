package raft

import (
	"context"
	"errors"
	"raft/proto/election"
	"time"

	"github.com/hkensame/goken/pkg/log"
)

/*
	考虑到整个raft各个模块之间绝大部分都会想要更改本节点的状态(包括term,fsm等),所以每次加锁时尽量控制到细密度,
	防止某个任务一直抢不到锁,同时也尽量在调用主函数前检查状态是否已经改变而没有条件再执行函数,
	后续可以考虑使用安全队列实现并发的串行化
*/

var (
	ErrInvalidStatus = errors.New("失效的状态,无法进行之后的逻辑")
)

func (r *Raft) electionTicker() {
	//先记录选举请求的初始状态
	for !r.closed {
		//如果是leader就休眠,不进行下面的检查
		if r.smustLock(LockStatus(Leader)) {
			r.sunlock()
			r.resetElectionTicker()
			time.Sleep(r.getSleepTime())
			continue
		}

		//先判断是否过了一个term周期
		if r.checkTimeout(r.electionTime) {
			log.Infof("节点%s经过一次term周期timeout", r.selfInfo.Id)
			r.slock()
			r.resetTerm(int(r.currentTerm + 1))
			r.voteFor = r.selfInfo.Id
			r.totalTickets = 1
			r.event(r.ctx, EventTimeout)
			r.sunlock()
			r.resetElectionTicker()
		}

		//如果是candidator就发起一轮投票请求
		if r.smustLock(LockStatus(Candidator)) {
			log.Infof("节点%s发起一轮投票请求,选举状态为: term:%d", r.selfInfo.Id, r.currentTerm)
			term := int(r.currentTerm)
			go r.requestVote(term)
			r.sunlock()
			time.Sleep(r.getSleepTime())
		} else {
			//如果不是candidator就休眠
			log.Infof("节点%s已不再是candidator,进入休眠", r.selfInfo.Id)
			time.Sleep(r.getSleepTime())
		}
	}
}

func (r *Raft) requestVote(term int) error {
	req := &election.ReceiveVoteReq{}
	//检查条件,是否是candidator并且term是否发生变化
	if r.smustLock(LockStatus(Candidator), LockEqualTerm(int(term))) {
		req.CandidateTerm = r.currentTerm
		req.CandidateId = r.selfInfo.Id
		req.LastLogginIndex = int32(len(r.persister.entries) - 1)
		req.LastLogginTerm = int32(r.persister.entries[req.LastLogginIndex].Term)
		r.sunlock()
	} else {
		return ErrInvalidStatus
	}

	for k, v := range r.clients {
		if !r.smustLock(LockStatus(Candidator), LockEqualTerm(int(term))) {
			return ErrInvalidStatus
		}

		//先看选票是不是已经达到成为leader的标准了,这里要调用变量leader的逻辑
		if r.totalTickets >= r.getMajorityNumber() {
			log.Infof("节点%s已经达到成为leader的标准,成为leader", r.selfInfo.Id)
			r.resetTerm(int(r.currentTerm))
			go r.replicationTicker()
			r.event(r.ctx, EventElected)
			r.sunlock()
			r.resetElectionTicker()
			//当选leader后会初始化节点的nextIndex和matchIndex,
			//其中leader乐观认为所有节点都是同步的,所以会将nextIndex初始化为与自己一样的日志序号
			for k := range r.clients {
				r.nextIndex[k.Id] = len(r.persister.entries)
				r.matchIndex[k.Id] = 0
			}
			return nil
		}

		//如果已经请求过则忽视
		if _, ok := r.ticketsSource[k.Id]; ok {
			r.sunlock()
			continue
		}

		cli, err := v.Dial()
		if err != nil {
			r.sunlock()
			//log.Errorf("raft对端不可达,对端信息为:%s, err = %v", v.Endpoint.String(), err)
			continue
		}
		res, err := election.NewElectionClient(cli).ReceiveVote(r.ctx, req)
		if err != nil {
			r.sunlock()
			//log.Errorf("raft调用对端ReceiveVote函数失败,对端信息为:%s, err = %v", v.Endpoint.String(), err)
			continue
		}

		// 如果收到的res是自身之前term发送的请求应当舍去,
		// 这可以在res中冗余一个candidatorTerm来识别
		if res.CandidatorTerm < r.currentTerm {
			r.sunlock()
			continue
		}

		// 如果raft节点发现自己的term小于其它节点的term,会立马更新自己的term并降为follower,
		// 随后将自己得到的选票清零,重置自己的选票
		// 重置自己的tiemeout等信息进入下一个选举周期,这里无论如何
		if res.VoterTerm > r.currentTerm {
			log.Infof("节点%s从来自%s的投票res中发现比自己大的term", r.selfInfo.Id, k.Id)
			r.resetElectionTicker()
			r.resetTerm(int(res.VoterTerm))
			r.event(r.ctx, EventLessTerm)
			r.sunlock()
			return nil
		} else {
			//无论对端是否投了票,只要成功连接且term不大于自身,之后都不再尝试请求对端
			r.ticketsSource[k.Id] = int(res.VoterTerm)
			if res.VoteFor {
				log.Infof("节点%s从来自%s的投票res中得到一票", r.selfInfo.Id, k.Id)
				r.totalTickets++
			}
		}
		r.sunlock()
	}
	return nil
}

// rpc函数
func (r *Raft) ReceiveVote(ctx context.Context, in *election.ReceiveVoteReq) (*election.ReceiveVoteRes, error) {
	res := &election.ReceiveVoteRes{
		VoterTerm: in.CandidateTerm,
		VoteFor:   false,
		//用于实现集票的幂等性
		CandidatorTerm: in.CandidateTerm,
	}

	r.slock()
	//flag为true表示candidator的日志新于自己
	flag := in.LastLogginTerm > int32(r.getLastTerm())
	flag = flag || (in.LastLogginTerm == int32(r.getLastTerm()) && in.LastLogginIndex >= int32(r.getLastIndex()))

	//如果候选者的请求中term要小于自身的term则不投票
	if in.CandidateTerm < r.currentTerm {
		res.VoterTerm = r.currentTerm
	} else if !flag {
		//如果candidator日志不如自己新会进入这个空else
	} else if in.CandidateTerm > r.currentTerm {
		r.voteForCandidator(in.CandidateId)
		r.resetTerm(int(in.CandidateTerm))
		r.event(r.ctx, EventLessTerm)
		res.VoteFor = true
	} else {
		if r.roleFsm.Current() == Leader {
		} else if r.voteFor == "" {
			r.voteForCandidator(in.CandidateId)
			res.VoteFor = true
		}
	}
	r.sunlock()
	return res, nil
}

func (r *Raft) voteForCandidator(id string) {
	r.voteFor = id
	r.resetElectionTicker()
}
