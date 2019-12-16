package main

import (
	"context"
	tcp "github.com/tevino/tcp-shaker"
	"log"
	"sync"
	"time"
)

var (
	ips  = []string{"127.0.0.1", "192.168.0.1", "10.5.6.7"}
	port = "80"
	mu   sync.RWMutex
)

func Scanner() {
	var addresses = make([]string, 0)
	var notReadyAddress = make([]string, 0)
	concurrency := len(ips)

	var wg sync.WaitGroup
	wg.Add(concurrency)
	//并发启动扫描函数
	for i := 0; i < concurrency; i++ {
		go ScanPort(ips[i], &addresses, &notReadyAddress, &wg)
	}

	// 等待执行完成
	wg.Wait()
	log.Printf("notReadyAddress: %v\n", notReadyAddress)
	log.Printf("Addresses: %v\n", addresses)
}

func ScanPort(ip string, addresses *[]string, notReadyAddress *[]string, wg *sync.WaitGroup) {
	log.Println("scaning ", ip, "port", port)

	c := tcp.NewChecker()

	ctx, stopChecker := context.WithCancel(context.Background())
	defer stopChecker()
	go func() {
		if err := c.CheckingLoop(ctx); err != nil {
			log.Println("checking loop stopped due to fatal error: ", err)
		}
	}()

	<-c.WaitReady()

	err := c.CheckAddr(ip+":"+port, time.Millisecond*1000)

	switch err {
	case tcp.ErrTimeout:
		log.Println("notReadyAddress: ", ip, "errMsg: ", err)
		mu.Lock()
		defer mu.Unlock()
		*notReadyAddress = append(*notReadyAddress, ip)
	case nil:
		log.Println("Addresses: ", ip)
		mu.Lock()
		defer mu.Unlock()
		*addresses = append(*addresses, ip)
	default:
		log.Println("Error occurred while connecting: ", err)
	}

	wg.Done()
}

func main() {
	for {
		Scanner()
		time.Sleep(1 * time.Second)
	}
}
