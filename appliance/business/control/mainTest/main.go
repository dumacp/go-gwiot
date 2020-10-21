package main

import (
	"log"
	"time"
)

func main() {

	count1 := 0
	count2 := 0
	chQuit := make(chan int, 0)
	fun1 := func() {
		go func() {
			for {
				select {
				case <-chQuit:
				default:
					log.Printf("count1: %v", count1)
					time.Sleep(1 * time.Second)
					count1++
				}
			}
		}()
		time.Sleep(5 * time.Second)
	}
	fun2 := func() {
		go func() {
			for {
				select {
				case <-chQuit:
				default:
					log.Printf("count2: %v", count2)
					time.Sleep(1 * time.Second)
					count2++
				}
			}
		}()
		time.Sleep(5 * time.Second)
	}
	fun1()
	fun2()
	for {
		time.Sleep(1 * time.Second)
		if count1 > 12 {
			close(chQuit)
			break
		}
	}
	time.Sleep(5 * time.Second)
}
