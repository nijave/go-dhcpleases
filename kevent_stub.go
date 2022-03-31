//+build linux

package main

import "time"

func (k KeventWatch) Watch(block bool) <-chan int {
	ch := make(chan int)

	go func() {
		for {
			ch <- 1
			time.Sleep(10 * time.Second)
		}
	}()

	return ch
}
