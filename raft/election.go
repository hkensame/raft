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

func (r *Raft) electionTicker(ctx context.Context) {
	//先记录选举请求的初始状态
	for !r.closed {
		//如果是leader就休眠,不进行下面的检查
		if r.smustLock(LockStatus(Leader)) {
			r.sunlock()
			r.tmtx.Lock()
			r.resetElectionTicker()
			r.tmtx.Unlock()
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
			r.event(ctx, EventTimeout)
			r.sunlock()

			r.tmtx.Lock()
			r.resetElectionTicker()
			r.tmtx.Unlock()

		}

		//如果是candidator就发起一轮投票请求
		if r.smustLock(LockStatus(Candidator)) {
			log.Infof("节点%s的选举状态为: term:%d", r.selfInfo.Id, r.currentTerm)
			log.Infof("节点%s发起一轮投票请求", r.selfInfo.Id)
			term := int(r.currentTerm)
			go r.requestVote(ctx, term)
			r.sunlock()
			time.Sleep(r.getSleepTime())
		} else {
			//如果不是candidator就休眠
			log.Infof("节点%s已不再是candidator,进入休眠", r.selfInfo.Id)
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(r.getSleepTime())
			}
		}
	}
}

func (r *Raft) requestVote(ctx context.Context, term int) error {
	req := &election.ReceiveVoteReq{}
	//检查条件,是否是candidator并且term是否发生变化
	if r.smustLock(LockStatus(Candidator), LockEqualTerm(int(term))) {
		req.CandidateTerm = r.currentTerm
		req.CandidateId = r.selfInfo.Id
		r.sunlock()
	} else {
		log.Infof("节点%s在投票请求中发现自己的term或状态已改变,退出请求", r.selfInfo.Id)
		return ErrInvalidStatus
	}

	for k, v := range r.clients {
		if !r.smustLock(LockStatus(Candidator), LockEqualTerm(int(term))) {
			log.Infof("节点%s在投票请求中发现自己的term或状态已改变,退出请求", r.selfInfo.Id)
			return ErrInvalidStatus
		}

		//先看选票是不是已经达到成为leader的标准了,这里要调用变量leader的逻辑
		if r.totalTickets >= getMajorityNumber(r.RaftNodesNumber) {
			log.Infof("节点%s已经达到成为leader的标准,成为leader", r.selfInfo.Id)
			r.resetTerm(int(r.currentTerm))
			go r.replicationTicker(ctx)
			r.event(ctx, EventElected)
			r.sunlock()
			r.tmtx.Lock()
			r.resetElectionTicker()
			r.tmtx.Unlock()
			//当选leader后会初始化节点的nextIndex和matchIndex,
			//其中leader乐观认为所有节点都是同步的,所以会将nextIndex初始化为与自己一样的日志序号
			for k := range r.clients {
				r.nextIndex[k.Id] = len(r.presister.entries)
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
			log.Errorf("raft对端不可达,对端信息为:%s, err = %v", v.Endpoint.String(), err)
			continue
		}
		res, err := election.NewElectionClient(cli).ReceiveVote(ctx, req)
		if err != nil {
			r.sunlock()
			log.Errorf("raft调用对端ReceiveVote函数失败,对端信息为:%s, err = %v", v.Endpoint.String(), err)
			continue
		}

		//如果收到的res是自身之前term发送的请求应当舍去,这可以在res中冗余一个candidatorTerm来识别
		if res.CandidatorTerm < r.currentTerm {
			r.sunlock()
			continue
		}

		//如果raft节点发现自己的term小于其它节点的term,会立马更新自己的term并降为follower,随后将自己得到的选票清零,重置自己的选票
		//重置自己的tiemeout等信息进入下一个选举周期,这里无论如何
		if res.VoterTerm > r.currentTerm {
			log.Infof("节点%s从来自%s的投票res中发现比自己大的term", r.selfInfo.Id, k.Id)
			r.tmtx.Lock()
			r.resetElectionTicker()
			r.tmtx.Unlock()
			r.resetTerm(int(res.VoterTerm))
			r.event(ctx, EventLessTerm)
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
	//如果自己的commit log index大于请求者的commit log index,则哪怕它的term大于自己也不会投票
	// if in.CommitIndex < int32(r.presister.commitIndex) {
	// 	return res, nil
	// }

	//如果候选者的请求中term要小于自身的term则不投票
	if in.CandidateTerm < r.currentTerm {
		res.VoterTerm = r.currentTerm
	} else if in.CandidateTerm > r.currentTerm {
		log.Infof("节点%s从来自节点%s的投票req中发现比自己大的term", r.selfInfo.Id, in.CandidateId)
		//如果候选者的term大于自身则更新自己的term,timeout,如果自身为candidator则清空candidator内的信息
		r.voteForCandidator(in.CandidateId)
		r.resetTerm(int(in.CandidateTerm))
		r.event(ctx, EventLessTerm)
		res.VoteFor = true
	} else {
		//如果term相同则若自己不是leader且未投过票则投给对端
		//这套函数保证了candidator和同term的leader不会进入这个逻辑
		if r.roleFsm.Current() == Leader {

		} else if r.voteFor == "" {
			log.Infof("节点%s在来自节点%s的投票req中发现与自己相当的term,且自身还未投票", r.selfInfo.Id, in.CandidateId)
			r.voteForCandidator(in.CandidateId)
			res.VoteFor = true
		} else {
			// 其次比较日志的新旧
			flag := in.LastLogginTerm > int32(r.lastLogginTerm)
			flag = flag || (in.LastLogginTerm == int32(r.lastLogginTerm) && in.LastLogginIndex >= int32(r.lastLogginIndex))
			if flag {
				r.voteForCandidator(in.CandidateId)
				res.VoteFor = true
			}
		}
	}
	r.sunlock()
	return res, nil
}

func (r *Raft) voteForCandidator(id string) {
	r.voteFor = id
	r.tmtx.Lock()
	r.resetElectionTicker()
	r.tmtx.Unlock()
}
