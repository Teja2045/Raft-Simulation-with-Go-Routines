package main

import (
	"fmt"
	"time"
)

func main1() {
	n := 1
	for {
		switch n {
		case 1:
			fmt.Println("val = ", n)
			n = 2
		case 2:
			fmt.Println(n)
			n = 3

		case 3:
			fmt.Println("sleep", n)
			go func() {
				time.Sleep(100000)
				n = 10
				fmt.Println("routine", n)
			}()
			n = 1
			fmt.Println("awoke", n)

		}
		if n == 10 {
			break
		}
		fmt.Println("end?", n)
	}
	fmt.Println("end")
}
