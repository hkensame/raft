package raft

import (
	//	"bytes"

	"context"
	"sync"
	"time"

	//	"course/labgob"

	"raft/proto/election"
	"raft/proto/replication"

	"github.com/hkensame/goken/pkg/common/hostgen"
	"github.com/hkensame/goken/pkg/log"
	"github.com/hkensame/goken/server/httpserver"
	"github.com/hkensame/goken/server/rpcserver"
	"github.com/looplab/fsm"
	"github.com/oklog/run"
)

const (
	electionTimeoutMin    time.Duration = 1000 * time.Millisecond
	electionTimeoutMax    time.Duration = 1500 * time.Millisecond
	replicationTimeoutMin time.Duration = 50 * time.Millisecond
	replicationTimeoutMax time.Duration = 100 * time.Millisecond
	sleepTimeMin          time.Duration = 250 * time.Millisecond
	sleepTimeMax          time.Duration = 400 * time.Millisecond
)

// const (
// 	electionTimeoutMax    time.Duration = 4 * time.Second
// 	electionTimeoutMin    time.Duration = 3 * time.Second
// 	replicationTimeoutMax time.Duration = 2 * time.Second
// 	replicationTimeoutMin time.Duration = 1 * time.Second
// 	sleepTimeMax          time.Duration = 2 * time.Second
// 	sleepTimeMin          time.Duration = 1 * time.Second
// )

const (
	Leader     = "leader"
	Candidator = "candidator"
	Follower   = "follower"
)

const (
	//节点本身周期结束
	EventTimeout = "timeout"
	//周期结束,Leader退为Follower
	EventExpire = "expire"
	//Leader宕机
	EventShotdown = "shotdown"
	//发现集群缺少Leader,可以开始选举
	EventElection = "election"
	//选举结束,该节点成功当选Leader
	EventElected = "elected"
	//选举结束,该节点未当选成功
	EventDisElected = "diselected"
	//Leader心跳超时,Follower变为Candidator
	EventHeartbeatTimeout = "heartbeat-timeout"
	//节点收到了其他大于自己的term,此时会将自己降为follower
	EventLessTerm = "less-term"
	//节点收到了来自leader的呼叫(这个呼叫包括选举,日志复制等)
	EventLeaderCall = "leader-call"
)

type Instance struct {
	Id   string
	Name string
	Host string
}

type Raft struct {
	ctx      context.Context
	clients  map[Instance]*rpcserver.Client
	server   *rpcserver.Server
	selfInfo *Instance
	closed   bool

	//资源锁
	smtx          sync.Mutex
	roleFsm       *fsm.FSM
	currentTerm   int32
	voteFor       string
	totalTickets  int
	ticketsSource map[string]int

	presister  *persister
	nextIndex  map[string]int
	matchIndex map[string]int

	//这两个属性用在选举中,只有以下情景会改变其值:
	//1.在收到来自leader的日志同步时更新
	//2.在作为leader收到client的request时更新
	lastLogginIndex int
	lastLogginTerm  int

	//raft会开启一个http服务接收客户端的请求
	httpServer *httpserver.Server
	httpChan   chan *replication.Entry

	//时间锁
	tmtx sync.Mutex
	//下一次进行选举的时间
	electionTime time.Time

	hasLeader bool
	leaderId  string

	//记录了该raft节点认为存在的可达的集群节点总数,包括自己
	//无论如何,只要raft存在,这个参数一定大于0
	RaftNodesNumber int

	election.UnimplementedElectionServer
	replication.UnimplementedReplicationServer
}

func NewRaftFsm() *fsm.FSM {
	return fsm.NewFSM(
		Follower, //初始状态设为Follower
		fsm.Events{
			//当Leader心跳超时时,Follower变为Candidator
			{Name: EventHeartbeatTimeout, Src: []string{Follower}, Dst: Candidator},
			//周期结束,非leader节点转为candidator进行选举
			{Name: EventTimeout, Src: []string{Candidator, Follower}, Dst: Candidator},
			//选举失败,Candidate变回Follower
			{Name: EventDisElected, Src: []string{Candidator}, Dst: Follower},
			//选举成功,Candidate变为Leader
			{Name: EventElected, Src: []string{Candidator}, Dst: Leader},
			//Leader发生周期性超时(任期结束),自动降级为Follower
			{Name: EventExpire, Src: []string{Leader}, Dst: Follower},
			//Leader崩溃(可能用于显式触发,如优雅关闭)
			{Name: EventShotdown, Src: []string{Leader}, Dst: Follower},
			//发现Leader丢失,Follower主动发起选举
			{Name: EventElection, Src: []string{Follower}, Dst: Candidator},
			//candidator在选举中发现自己的term并非最新时更新当前term并将自己身份降为follower
			{Name: EventLessTerm, Src: []string{Candidator, Follower, Leader}, Dst: Follower},
			//收到任期正确(大于自身)的leader的call,无论自己是什么身份都变为follower
			{Name: EventLeaderCall, Src: []string{Candidator, Follower, Leader}, Dst: Follower},
		},
		fsm.Callbacks{},
	)
}

func MustNewRaft(ctx context.Context, id string, bind string, opts ...RaftOption) *Raft {
	r := &Raft{
		ctx:             ctx,
		closed:          false,
		roleFsm:         NewRaftFsm(),
		currentTerm:     1,
		voteFor:         "",
		clients:         make(map[Instance]*rpcserver.Client),
		RaftNodesNumber: 1,
		totalTickets:    0,
		ticketsSource:   make(map[string]int),
		selfInfo:        new(Instance),
		presister:       mustNewPersister(),
		nextIndex:       make(map[string]int),
		matchIndex:      make(map[string]int),
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.selfInfo.Name == "" {
		r.selfInfo.Name = r.selfInfo.Id
	}
	if ok := hostgen.ValidListenHost(bind); !ok {
		panic("非可用可绑定的host地址")
	}
	r.selfInfo.Host = bind

	r.server = rpcserver.MustNewServer(ctx,
		rpcserver.WithHost(bind),
		rpcserver.WithServiceID(id),
		rpcserver.WithServiceName(r.selfInfo.Name),
	)

	//插入一条默认的term数据,这有利于后面左边界的判断
	r.presister.entries = append(r.presister.entries, &replication.Entry{Term: 0, Command: nil})

	election.RegisterElectionServer(r.server.Server, r)
	replication.RegisterReplicationServer(r.server.Server, r)
	for _, v := range r.clients {
		_, err := v.Dial()
		if err != nil {
			panic(err)
		}
	}
	return r
}

func (r *Raft) Serve() {
	ctx, cancel := context.WithCancel(r.ctx)
	g := &run.Group{}

	g.Add(
		func() error {
			r.resetElectionTicker()
			once := &sync.Once{}
			for !r.closed {
				once.Do(
					func() {
						go r.electionTicker(ctx)
					},
				)
				select {
				case <-ctx.Done():
					return nil
				default:
					log.Infof("节点%s的状态为: term:%d,status:%s,has_leader:%t", r.selfInfo.Id, r.currentTerm, r.roleFsm.Current(), r.hasLeader)
					time.Sleep(3 * time.Second)
				}
			}
			cancel()
			r.closed = true
			return nil
		},
		func(err error) {
			cancel()
		},
	)

	g.Add(
		func() error {
			if err := r.server.Serve(); err != nil {
				log.Errorf("[raft] raft服务启动失败 err = %v", err)
				return err
			}
			return nil
		},
		func(err error) {
			r.server.GracefulStop()
		},
	)

	g.Run()

}

func (r *Raft) Close() {
	r.closed = true
}
