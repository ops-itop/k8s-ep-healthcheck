package main

import (
	"encoding/json"
	"log"
	//"reflect"
	"fmt"
	"net"
	"sync"
	"time"

	//"k8s.io/apimachinery/pkg/api/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var mu sync.Mutex

// global var store all endpoints
var ep []corev1.Endpoints

func _init(c *kubernetes.Clientset, l metav1.ListOptions) {
	endpoints, err := c.CoreV1().Endpoints("").List(l)
	if err != nil {
		panic(err.Error())
	}
	mu.Lock()
	ep = endpoints.Items
	mu.Unlock()
	//epStr, _ := json.MarshalIndent(ep, "", " ")
	//log.Printf("Endpionts: %s\n", epStr)
}

func watchEndpoints(c *kubernetes.Clientset, l metav1.ListOptions) {

}

// patch endpoint
func update(c *kubernetes.Clientset, namespace string, epName string, data map[string]interface{}) {
	playLoadBytes, _ := json.Marshal(data)

	_, err := c.CoreV1().Endpoints(namespace).Patch(epName, types.StrategicMergePatchType, playLoadBytes)

	if err != nil {
		log.Printf("Update Ednpoint %v.%v Error: %v", namespace, epName, err)
	}

	log.Printf("New addresses for Endpoint %v.%v: %v", namespace, epName, string(playLoadBytes))
}

// convert ip list to endpoints addresses list
func addrBuilder(addrs []string) []interface{} {
	addrList := make([]interface{}, 0)

	for _, v := range addrs {
		item := map[string]string{"ip": v}
		addrList = append(addrList, item)
	}

	return addrList
}

// build new endpoints subsets
func epBuilder(addresses []string, notReadyAddresses []string, ports []corev1.EndpointPort) map[string]interface{} {
	addr := make(map[string]interface{})
	subsets := make([]interface{}, 0)
	item := make(map[string]interface{})

	item["notReadyAddresses"] = addrBuilder(notReadyAddresses)
	item["addresses"] = addrBuilder(addresses)
	item["ports"] = ports

	subsets = append(subsets, item)
	addr["subsets"] = subsets

	return addr
}

// check if two slice equal
func StringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	if (a == nil) != (b == nil) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

// tcp checker
func tcpChecker(e corev1.Endpoints, c *kubernetes.Clientset, pwg *sync.WaitGroup) {
	ips := make([]string, 0)
	notReadyIps := make([]string, 0)
	var port string

	for _, v := range e.Subsets[0].Addresses {
		ips = append(ips, v.IP)
	}

	for _, v := range e.Subsets[0].NotReadyAddresses {
		ips = append(ips, v.IP)
		notReadyIps = append(notReadyIps, v.IP)
	}

	// 只支持检测第一个端口
	port = fmt.Sprint(e.Subsets[0].Ports[0].Port)
	if port == "" {
		return
	}

	var addresses = make([]string, 0)
	var notReadyAddresses = make([]string, 0)
	concurrency := len(ips)

	var wg sync.WaitGroup
	wg.Add(concurrency)
	//并发启动扫描函数
	for i := 0; i < concurrency; i++ {
		go checkPort(ips[i], port, &addresses, &notReadyAddresses, &wg)
	}

	// 等待执行完成
	wg.Wait()
	log.Printf("notReadyAddresses: %v\n", notReadyAddresses)
	log.Printf("notReadyIps: %v\n", notReadyIps)
	log.Printf("Addresses: %v\n", addresses)

	addr := epBuilder(addresses, notReadyAddresses, e.Subsets[0].Ports)
	if len(addresses) > 0 {
		if StringSliceEqual(notReadyIps, notReadyAddresses) {
			log.Printf("Already Marked notReady IPs for %v.%v. Ignore", e.Namespace, e.Name)
		} else {
			update(c, e.Namespace, e.Name, addr)
		}
	} else {
		log.Printf("No lived endpoints in %v.%v. Ignore", e.Namespace, e.Name)
	}

	pwg.Done()
}

// do check
func checkPort(ip string, port string, addresses *[]string, notReadyAddresses *[]string, wg *sync.WaitGroup) {
	log.Println("scaning ", ip, "port", port)

	err := retryPort(ip, port)

	if err != nil {
		log.Println("notReadyAddresses: ", ip, "errMsg: ", err)
		mu.Lock()
		*notReadyAddresses = append(*notReadyAddresses, ip)
		mu.Unlock()
	} else {
		*addresses = append(*addresses, ip)
	}

	wg.Done()
}

// retry
func retryPort(ip string, port string) error {
	var e error
	for i := 0; i < 3; i++ {
		conn, err := net.DialTimeout("tcp", ip+":"+port, time.Millisecond*100)
		if conn != nil {
			defer conn.Close()
		}

		if err == nil {
			return err
		} else {
			log.Printf("Dial %v:%v failed. will retry %v", ip, port, i)
			e = err
			time.Sleep(time.Millisecond * 100)
		}
	}
	return e
}

func main() {

	// only check custom endpoints with label type=external
	labelSelector := "type=external"
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		panic(err.Error())
	}

	// 首先初始化 ep 变量
	_init(clientset, listOptions)

	var wg sync.WaitGroup
	// 监视 ep 变更事件
	go watchEndpoints(clientset, listOptions)

	for {
		for _, e := range ep {
			wg.Add(1)
			// tcp检测
			go tcpChecker(e, clientset, &wg)
		}

		wg.Wait()
		time.Sleep(1 * time.Second)
	}
}
