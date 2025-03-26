package main

import (
	"context"
	"raft/raft"
	"time"

	"github.com/hkensame/goken/server/rpcserver"
)

func main() {
	ctx := context.Background()
	c := map[raft.Instance]*rpcserver.Client{}

	ep1 := raft.Instance{}
	ep1.Host = "192.168.199.128:20000"
	ep1.Id = "raft1"
	ep1.Name = "raft-node-1"
	c1 := rpcserver.MustNewClient(ctx, ep1.Host)

	ep2 := raft.Instance{}
	ep2.Host = "192.168.199.128:20001"
	ep2.Id = "raft2"
	ep2.Name = "raft-node-2"
	c2 := rpcserver.MustNewClient(ctx, ep2.Host)

	c[ep1] = c1
	c[ep2] = c2

	r := raft.MustNewRaft(ctx, "raft3", "0.0.0.0:20002", raft.WithClients(c), raft.WithRaftNodesNumber(3))

	time.Sleep(5 * time.Second)
	r.Serve()
}
