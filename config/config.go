package config

type RaftConfig struct {
	//客观定义的选举超时的最大时间,默认单位为Milisecond,每次raft节点都会随机一个在此范围内的值
	VoteTLEMax int `mapstructure:"vote_timeout_max"`
	//客观定义的选举超时的最小时间,默认单位为Milisecond
	VoteTLEMin int `mapstructure:"vote_timeout_min"`
	//rpc连接请求建立的超时时间,
	DialTimeout int `mapstructure:"dial_timeout"`
	//其他各节点的
	EndPointsHost []string `mapstructure:"endpoints_host"`
}
