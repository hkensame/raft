package raft

import (
	"context"
	"sync"
	"time"

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

const (
	Leader     = "leader"
	Candidator = "candidator"
	Follower   = "follower"
	Deader     = "deader"
)

const (
	//所有节点都是可用的
	Avaliable int = iota
	//存在节点宕机或不可用但是集群整体可用
	PartialAvailable
	//集群已经无法使用
	UnAvaliable
)

const (
	Follow int = iota
	Candidate
	Disconnect
)

const (
	//节点本身周期结束
	EventTimeout = "timeout"
	//Leader关闭
	EventShotdown = "shotdown"
	//选举结束,该节点成功当选Leader
	EventElected = "elected"
	//节点收到了其他大于自己的term,此时会将自己降为follower
	EventLessTerm = "less-term"
	//节点收到了来自leader的呼叫(这个呼叫包括选举,日志复制等)
	EventLeaderCall = "leader-call"
)

type Instance struct {
	Id       string
	Name     string
	Host     string
	HttpAddr string
	Status   int
}

type Raft struct {
	ctx      context.Context
	clients  map[*Instance]*rpcserver.Client
	server   *rpcserver.Server
	selfInfo *Instance
	closed   bool

	//资源锁
	smtx    sync.Mutex
	roleFsm *fsm.FSM
	//这个字段应当被持久化
	currentTerm   int32
	voteFor       string
	totalTickets  int
	ticketsSource map[string]int

	persister  *persister
	nextIndex  map[string]int
	matchIndex map[string]int

	//raft会开启一个http服务接收客户端的请求
	httpServer *httpserver.Server
	//这个httpChan负责从http服务端获取到收到的entry请求
	httpChan chan *replication.Entry

	//时间锁
	tmtx sync.Mutex
	//下一次进行选举的时间
	electionTime time.Time

	hasLeader bool
	leaderId  string

	//记录了该raft节点认为存在的可达的集群节点总数,包括自己
	//无论如何,只要raft存在,这个参数一定大于0
	raftNodesNumber int
	filePath        string
	election.UnimplementedElectionServer
	replication.UnimplementedReplicationServer
}

func NewRaftFsm() *fsm.FSM {
	return fsm.NewFSM(
		Follower, //初始状态设为Follower
		fsm.Events{
			//周期结束,非leader节点转为candidator进行选举
			{Name: EventTimeout, Src: []string{Candidator, Follower}, Dst: Candidator},
			//选举成功,Candidate变为Leader
			{Name: EventElected, Src: []string{Candidator}, Dst: Leader},
			//Leader崩溃(可能用于显式触发,如优雅关闭)
			{Name: EventShotdown, Src: []string{Leader, Candidator, Follower}, Dst: Deader},
			//candidator在选举中发现自己的term并非最新时更新当前term并将自己身份降为follower
			{Name: EventLessTerm, Src: []string{Candidator, Follower, Leader}, Dst: Follower},
			//收到任期正确(大于自身)的leader的call,无论自己是什么身份都变为follower
			{Name: EventLeaderCall, Src: []string{Candidator, Follower, Leader}, Dst: Follower},
		},
		fsm.Callbacks{},
	)
}

func MustNewRaft(ctx context.Context, id string, bind string, ch chan *replication.Entry, opts ...RaftOption) *Raft {
	r := &Raft{
		ctx:             ctx,
		closed:          false,
		roleFsm:         NewRaftFsm(),
		currentTerm:     1,
		voteFor:         "",
		clients:         make(map[*Instance]*rpcserver.Client),
		raftNodesNumber: 1,
		totalTickets:    0,
		ticketsSource:   make(map[string]int),
		selfInfo:        &Instance{Id: id},

		nextIndex:  make(map[string]int),
		matchIndex: make(map[string]int),
		filePath:   "./data1",
	}
	for _, opt := range opts {
		opt(r)
	}
	if ok := hostgen.ValidListenHost(bind); !ok {
		panic("非可用可绑定的host地址")
	}

	r.persister = mustNewPersister(r.filePath)
	r.persister.persistCond = sync.NewCond(&r.smtx)

	if r.selfInfo.Name == "" {
		r.selfInfo.Name = r.selfInfo.Id
	}
	r.selfInfo.Host = bind

	r.server = rpcserver.MustNewServer(ctx,
		rpcserver.WithHost(bind),
		rpcserver.WithServiceID(id),
		rpcserver.WithServiceName(r.selfInfo.Name),
	)

	election.RegisterElectionServer(r.server.Server, r)
	replication.RegisterReplicationServer(r.server.Server, r)
	for _, v := range r.clients {
		_, err := v.Dial()
		if err != nil {
			panic(err)
		}
	}

	r.loadMetadata()
	r.loadEntries()

	return r
}

func (r *Raft) Serve() {
	g := &run.Group{}

	g.Add(
		func() error {
			r.resetElectionTicker()
			once := &sync.Once{}
			for !r.closed {
				once.Do(
					func() {
						go r.electionTicker()
						go r.applicationTicker()
						go func() {
							for !r.closed {
								time.Sleep(30 * time.Second)
								r.persistMetadata()
							}
						}()
					},
				)
				log.Infof("节点%s的状态为: term:%d,status:%s,has_leader:%t", r.selfInfo.Id, r.currentTerm, r.roleFsm.Current(), r.hasLeader)
				time.Sleep(3 * time.Second)
			}
			return nil
		},

		func(err error) {
			r.Close()
			time.Sleep(3 * time.Second)
			r.roleFsm.SetState(Deader)
			log.Info("[raft] 节点正常关闭")
			r.persistMetadata()
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

		func(err error) {},
	)

	g.Run()

}

func (r *Raft) Close() {
	r.closed = true
}
