package main

import (
	"fmt"
	"time"
)

func main2() {
	period := time.Duration(5000) * time.Millisecond
	timeout := time.Duration(period)
	ch := make(chan bool, 10)
	chDummy := make(chan bool, 10)
	go func(ch chan bool) {
		for i := 0; i < 10; i++ {
			ch <- false
			time.Sleep(period / 2)
		}
	}(ch)
	exit := false

	go func(chDummy chan bool) {
		for {
			if exit {
				break
			}
			ok := <-ch
			if ok {
				chDummy <- true
			}
		}
	}(chDummy)
	for {
		fmt.Println("bool ", exit)
		if exit {
			break
		}
		select {
		case receive := <-chDummy:
			fmt.Println("receive")
			if receive {
				timeout = time.Duration(period)
				fmt.Println("receive pk")
			}
		case <-time.After(timeout):
			fmt.Println("exit")
			exit = true
		}
	}
	fmt.Println("done")
}
