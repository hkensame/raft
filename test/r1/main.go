package main

import (
	"context"
	"raft/raft"
	"time"

	"github.com/hkensame/goken/server/rpcserver"
)

func main() {
	c := map[raft.Instance]*rpcserver.Client{}

	ctx := context.Background()
	ep1 := raft.Instance{}
	ep1.Host = "192.168.199.128:20001"
	ep1.Id = "raft2"
	ep1.Name = "raft-node-2"
	c1 := rpcserver.MustNewClient(ctx, ep1.Host)

	ep2 := raft.Instance{}
	ep2.Host = "192.168.199.128:20002"
	ep2.Id = "raft3"
	ep2.Name = "raft-node-3"
	c2 := rpcserver.MustNewClient(ctx, ep2.Host)

	c[ep1] = c1
	c[ep2] = c2

	r := raft.MustNewRaft(ctx, "raft1", "0.0.0.0:20000", raft.WithClients(c), raft.WithRaftNodesNumber(3))

	time.Sleep(5 * time.Second)
	r.Serve()
}
