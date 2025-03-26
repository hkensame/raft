package ktime

import (
	"sync"
	"time"

	"github.com/hkensame/goken/pkg/log"
)

// 可重置定时器结构体
type ResettableTimer struct {
	mu      sync.Mutex
	timer   *time.Timer
	period  time.Duration
	resetCh chan struct{}
	//是否被关闭
	closed  bool
	fn      func() bool
	doFirst bool
	tfn     func() time.Duration
}

type OptionFunc func(*ResettableTimer)

// 创建新的可重置定时器,该定时器在不调用reset的情况下每隔一次指定period时间段都将运行一次fn,
// fn的类型是func()bool,若fn返回false则停止定时
// 如果调用了reset则重新进入等待而不是类似官方timer那样触发fn
func MustNewResettableTimer(period time.Duration, fn func() bool, opts ...OptionFunc) *ResettableTimer {
	rt := &ResettableTimer{
		period:  period,
		fn:      fn,
		closed:  true,
		resetCh: make(chan struct{}),
	}

	rt.tfn = func() time.Duration {
		return period
	}

	for _, opt := range opts {
		opt(rt)
	}

	return rt
}

func (rt *ResettableTimer) Run() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	//允许stop后重启,如果发现timer是关闭的则重新启动
	if rt.closed {
		rt.resetCh = make(chan struct{})
		rt.closed = false
		go rt.runLoop()
	}
}

// 运行定时器的主循环
func (rt *ResettableTimer) runLoop() {
	rt.mu.Lock()
	if !rt.closed && rt.doFirst && !rt.fn() {
		rt.closed = true
		rt.mu.Unlock()
		return
	}

	rt.mu.Unlock()
	rt.timer = time.NewTimer(rt.tfn())

	for !rt.closed {
		t := time.Now()
		log.Warn("开始一次timer等待")

		select {
		case <-rt.timer.C:
			log.Warnf("一次timer任务触发成功但未执行完成%v", time.Now().Sub(t))
			rt.mu.Lock()
			if rt.closed {
				rt.mu.Unlock()
				return
			}

			//执行一次定时任务
			if !rt.fn() {
				rt.closed = true
			} else {
				//如果运行fn成功则重新开始定时运行任务
				rt.timer.Stop()
				// 清除可能残留的事件
				select {
				case <-rt.timer.C:
				default:
				}
				rt.timer = time.NewTimer(rt.tfn())

				log.Warnf("一次timer最终执行完成%v", time.Now().Sub(t))
			}
			rt.mu.Unlock()

		case <-rt.resetCh:
			//如果reset函数触发则重置timer,老的timer直接忽视
			rt.mu.Lock()
			if rt.closed {
				rt.mu.Unlock()
				return
			}

			rt.timer.Stop()
			select {
			case <-rt.timer.C:
			default:
			}

			log.Warnf("一次timer被reset")

			rt.timer = time.NewTimer(rt.tfn())
			rt.mu.Unlock()
		}

	}
}

// 重新开始定时器
func (rt *ResettableTimer) Reset() {
	rt.ResetWithTTK(rt.period)
}

func (rt *ResettableTimer) ResetWithTTK(t time.Duration) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.closed {
		return
	}
	select {
	case rt.resetCh <- struct{}{}:
	default:
	}
}

// 停止定时器
func (rt *ResettableTimer) Stop() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if !rt.closed {
		rt.closed = true
		//关闭channel间接使已失效的runLoop关闭
		close(rt.resetCh)
	}
	rt.timer.Stop()
}

// 让整个定时器先执行一次传入的func再进行定时,注意如果关闭了定时器再开启会再次判断一次doFirst
func WithDoFirst(do bool) OptionFunc {
	return func(rt *ResettableTimer) {
		rt.doFirst = do
	}
}

// 允许让fn生成在时间区间内随机的等待时间
func WithRandomTimer(fn func() time.Duration) OptionFunc {
	return func(rt *ResettableTimer) {
		rt.tfn = fn
	}
}
