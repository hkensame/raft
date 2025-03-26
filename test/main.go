package main

import (
	"fmt"
	"time"
)

func main() {
	// t := ktime.MustNewResettableTimer(1*time.Second, func() bool {
	// 	fmt.Println("hello")
	// 	return true
	// })

	// t.Run()
	ch := make(chan struct{})

	go func() {
		select {
		case <-ch:
			fmt.Println("ch 收到信息")
			time.Sleep(1 * time.Minute)
		}
	}()
	select {
	case ch <- struct{}{}:
	default:
		fmt.Printf("ch已空d")
	}
	time.Sleep(1 * time.Second)
	select {
	case ch <- struct{}{}:
	default:
		fmt.Printf("ch已空")
	}

}
