package main

import (
	"log"
	"net"
	"sync"
	"time"
)

var ips = []string{"127.0.0.1", "192.168.0.1", "10.2.3.4", "10.5.6.7"}
var port = "80"
var mu sync.Mutex

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
	//log.Printf("Addresses: %v\n", addresses)
}

func ScanPort(ip string, addresses *[]string, notReadyAddress *[]string, wg *sync.WaitGroup) {
	//log.Println("scaning ", ip, "port", port)
	_, err := net.DialTimeout("tcp", ip+":"+port, time.Millisecond*100)

	if err != nil {
		//log.Println("notReadyAddress: ", ip, "errMsg: ", err)
		mu.Lock()
		*notReadyAddress = append(*notReadyAddress, ip)
		mu.Unlock()
	} else {
		*addresses = append(*addresses, ip)
	}

	wg.Done()
}

func main() {
	for {
		Scanner()
		time.Sleep(1 * time.Second)
	}
}
