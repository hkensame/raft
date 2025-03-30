package raft

import (
	"bufio"
	"context"
	"os"
	"raft/proto/replication"
	"sync"

	"github.com/hkensame/goken/pkg/log"
	"google.golang.org/protobuf/proto"
)

// 这个结构体中所有值字段都可被持久化,pendingIndex可以考虑不持久,而是通过commitedIndex和entries计算得出
type persister struct {
	mtx            sync.Mutex
	entries        []*replication.Entry
	commitedIndex  int
	persistedIndex int
	//待提交的index,这个index恒大于等于commitedIndex,但是commitedIndex是持久化的,而pendingIndex表示将要提交到的index
	pendingIndex int
	persistCond  *sync.Cond

	//内部应该是一个bufio.ReadWriter
	file *os.File
	// raftstate []byte
	// snapshot  []byte
}

func mustNewPersister(filepath string) *persister {
	p := &persister{
		entries: make([]*replication.Entry, 0),
	}
	var err error
	p.file, err = os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	return p
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

			//在go携程前加锁,在go携程结束后解锁能避免后触发的持久化任务先于先触发的持久化任务
			r.persister.mtx.Lock()
			go func() {
				if r.persister.persistence(entries) {
					r.slock()
					r.persister.commitedIndex += len(entries)
					r.sunlock()
				}
				r.persister.mtx.Unlock()
			}()
		}
	}
}

const blockSize = 1 << 12

// 持久化提交enties,只要把一个entry持久到磁盘就永久固定了一个commitedIndex,不用担心持久化时宕机
// TODO 在这个函数内既需要持久化entries也需要更新commitedIndex
// 先写一个带缓冲的试试水
func (p *persister) persistence(ent []*replication.Entry) bool {
	start, _ := p.file.Stat()
	initialSize := start.Size()
	writer := bufio.NewWriterSize(p.file, blockSize)

	for _, v := range ent {
		data, err := proto.Marshal(v)
		if err != nil {
			log.Errorf("格式化ent失败,格式化对象为:%v", v)
			p.file.Truncate(initialSize)
			return false
		}

		_, err = writer.Write(data)
		if err != nil {
			log.Errorf("写入数据失败,对象为:%v", v)
			p.file.Truncate(initialSize)
			return false
		}
	}

	err := writer.Flush()
	if err != nil {
		log.Errorf("刷新数据失败")
		p.file.Truncate(initialSize)
		return false
	}
	p.persistedIndex += len(ent)
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
