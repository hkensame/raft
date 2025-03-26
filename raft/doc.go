package raft

/*
	如果一个节点收到了其它一个节点(b节点)的票,则该节点同一term期间是不会再向b节点请求投票的(当然,要考虑网络延迟)

	关于幂等性:考虑到上述条件以及candidator更新term时会将自己的选票清空,这似乎能实现幂等性
*/

/*
	关于选举时term的改变规律:
	1.请求者发现voter的term大于自己,改term,但不会请求选举
	2.voter发现请求者的term大于自己,改term,也不会请求选举
	3.周期timeout,term+1,将自己身份改为candidator,会请求选举
*/
