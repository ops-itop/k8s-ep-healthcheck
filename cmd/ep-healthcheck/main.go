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
	"github.com/caarlos0/env/v6"
	"github.com/ops-itop/k8s-ep-healthcheck/pkg/notify/wechat"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var mu sync.Mutex

// global var store all endpoints
var ep []corev1.Endpoints

var wechatToken wechat.AccessToken

var listOptions metav1.ListOptions

type config struct {
	LabelSelector string `env:"LABELSELECTOR" envDefault:"type=external"` //only check custom endpoints with label type=external
	Touser        string `env:"TOUSER", envDefault:"@all"`
	Corpid        string `env:"CORPID"`
	Corpsecret    string `env:"CORPSECRET"`
	Agentid       int    `env:"AGENTID"`
}

var cfg config

func _init(c *kubernetes.Clientset) {
	endpoints, err := c.CoreV1().Endpoints("").List(listOptions)
	if err != nil {
		log.Fatal(err.Error())
	}
	mu.Lock()
	ep = endpoints.Items
	mu.Unlock()
	//epStr, _ := json.MarshalIndent(ep, "", " ")
	//log.Printf("Endpionts: %s\n", epStr)
}

// need update global var ep.
func watchEndpoints(c *kubernetes.Clientset) {
	watcher, err := c.CoreV1().Endpoints("").Watch(listOptions)
	if err != nil {
		log.Fatal(err.Error())
	}

	for e := range watcher.ResultChan() {
		endpoint := e.Object.(*corev1.Endpoints)
		log.Printf("Event %v on %v.%v. Re init", e.Type, endpoint.Namespace, endpoint.Name)
		_init(c)
	}
}

// patch endpoint
func update(c *kubernetes.Clientset, namespace string, epName string, data map[string]interface{}) {
	playLoadBytes, _ := json.Marshal(data)

	_, err := c.CoreV1().Endpoints(namespace).Patch(epName, types.StrategicMergePatchType, playLoadBytes)

	if err != nil {
		log.Printf("Update Ednpoint %v.%v Error: %v", namespace, epName, err)
		return
	}

	log.Printf("New addresses for Endpoint %v.%v: %v", namespace, epName, string(playLoadBytes))

	// notify
	err = wechat.UpdateToken(&wechatToken, cfg.Corpid, cfg.Corpsecret)
	if err != nil {
		log.Printf("Notify error. UpdateToken failed. Endpoint %v.%v: %v", namespace, epName, err)
		return
	}

	content := "Custom Endpoint HealthCheck:\nNew address for Endpoint " + namespace + "." + epName + "\n" + string(playLoadBytes)
	msg := wechat.WechatMsg{Touser: cfg.Touser, Msgtype: "text", Agentid: cfg.Agentid, Text: map[string]string{"content": content}}
	buf, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Notify error. json.Marshal(msg) failed. Endpoint %v.%v: %v", namespace, epName, err)
		return
	}
	err = wechat.SendMsg(wechatToken.Access_token, buf)
	if err != nil {
		log.Printf("Notify error. SendMsg failed. Endpoint %v.%v: %v", namespace, epName, err)
		return
	}
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

// check if string in slice
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// check if two slice equal
func StringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	if (a == nil) != (b == nil) {
		return false
	}

	// 忽略顺序
	for _, v := range a {
		if !contains(b, v) {
			return false
		}
	}

	return true
}

func getIPs(e corev1.Endpoints) ([]string, []string) {
	ips := make([]string, 0)
	notReadyIps := make([]string, 0)

	for _, v := range e.Subsets[0].Addresses {
		ips = append(ips, v.IP)
	}

	for _, v := range e.Subsets[0].NotReadyAddresses {
		ips = append(ips, v.IP)
		notReadyIps = append(notReadyIps, v.IP)
	}
	return ips, notReadyIps
}

// tcp checker
func tcpChecker(e corev1.Endpoints, c *kubernetes.Clientset, pwg *sync.WaitGroup) {
	ips, notReadyIps := getIPs(e)
	var port string

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
	//log.Printf("Addresses: %v\n", addresses)

	addr := epBuilder(addresses, notReadyAddresses, e.Subsets[0].Ports)
	if len(addresses) > 0 {
		if StringSliceEqual(notReadyIps, notReadyAddresses) {
			log.Printf("Already Marked notReady IPs for %v.%v. Ignore", e.Namespace, e.Name)
		} else {
			log.Printf("notReadyAddresses: %v\n", notReadyAddresses)
			log.Printf("notReadyIps: %v\n", notReadyIps)
			// 执行更新前有必要看看线上endpoints是否和 ips 完全一致，防止出现老数据刷掉新数据的情况
			currentEp, err := c.CoreV1().Endpoints(e.Namespace).Get(e.Name, metav1.GetOptions{})
			if err != nil {
				return
			}
			currentIPs, _ := getIPs(*currentEp)

			if StringSliceEqual(ips, currentIPs) {
				update(c, e.Namespace, e.Name, addr)
			} else {
				log.Printf("currentIps not same with ips for %v.%v. Ignore", e.Namespace, e.Name)
				// update global ep
				_init(c)
			}
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
		mu.Lock()
		*addresses = append(*addresses, ip)
		mu.Unlock()
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

	// app config
	err = env.Parse(&cfg)
	if err != nil {
		panic(err.Error())
	}

	listOptions.LabelSelector = cfg.LabelSelector

	// 首先初始化 ep 变量
	_init(clientset)

	var wg sync.WaitGroup
	// 监视 ep 变更事件
	go watchEndpoints(clientset)

	for {
		if len(ep) == 0 {
			log.Println("no custom endpoints.")
		}
		for _, e := range ep {
			wg.Add(1)
			// tcp检测
			go tcpChecker(e, clientset, &wg)
		}

		wg.Wait()
		time.Sleep(1 * time.Second)
	}
}
