package main

import (
	"encoding/json"
	//"reflect"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	//"k8s.io/apimachinery/pkg/api/errors"
	"github.com/caarlos0/env/v6"
	"github.com/ops-itop/k8s-ep-healthcheck/pkg/notify/wechat"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// version info
var (
	gitHash   string
	version   string
	buildTime string
	goVersion string
)

type config struct {
	LabelSelector string `env:"LABELSELECTOR" envDefault:"type=external"` //only check custom endpoints with label type=external
	Touser        string `env:"TOUSER", envDefault:"@all"`
	Corpid        string `env:"CORPID"`
	Corpsecret    string `env:"CORPSECRET"`
	Agentid       int    `env:"AGENTID"`
	LogLevel      string `env:"LOGLEVEL" envDefault:"debug"`
}

// one ipaddress for scaning
type ipaddress struct {
	Namespace string
	Name      string
	Ipaddress string
	Port      string
}

var (
	clientset   *kubernetes.Clientset
	cfg         config
	mu          sync.Mutex
	ep          []corev1.Endpoints // store all endpoints
	wechatToken wechat.AccessToken
	listOptions metav1.ListOptions //labelSelector for endpoints
)

func logInit() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{TimestampFormat: time.RFC3339, FullTimestamp: true})
	logLevel, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Panic("Log Level illegal.You should use trace,debug,info,warn,warning,error,fatal,panic")
	}
	log.SetLevel(logLevel)
}

func k8sClientInit() {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Panic(err.Error())
	}
	// creates the clientset
	clientset, err = kubernetes.NewForConfig(config)

	if err != nil {
		log.Panic(err.Error())
	}
}

// get all endpoints with labelSelector
func getEndpoints() {
	endpoints, err := clientset.CoreV1().Endpoints("").List(listOptions)
	if err != nil {
		log.Fatal("Init endpoints error. ", err.Error())
	}
	mu.Lock()
	ep = endpoints.Items
	mu.Unlock()
	log.Info("Init endpoints seccessful")
	epStr, _ := json.MarshalIndent(ep, "", " ")
	log.Trace("Endpionts: ", string(epStr))
}

// need update global var ep.
func watchEndpoints() {
	watcher, err := clientset.CoreV1().Endpoints("").Watch(listOptions)
	if err != nil {
		log.Fatal("Watch endpoints error. ", err.Error())
	}

	for e := range watcher.ResultChan() {
		endpoint := e.Object.(*corev1.Endpoints)
		log.WithFields(log.Fields{
			"namespace": endpoint.Namespace,
			"endpoint":  endpoint.Name,
		}).Info("Event ", e.Type, " watched. Re init.")
		getEndpoints()
	}
}

// patch endpoint
func patchEndpoint(namespace string, epName string, data map[string]interface{}) {
	epLog := log.WithFields(log.Fields{
		"namespace": namespace,
		"endpoint":  epName,
	})

	playLoadBytes, _ := json.Marshal(data)

	_, err := clientset.CoreV1().Endpoints(namespace).Patch(epName, types.StrategicMergePatchType, playLoadBytes)

	if err != nil {
		epLog.Error("Patch Ednpoint Error: ", err.Error())
		return
	}

	epLog.Warn("Patch Endpoint Succ: ", string(playLoadBytes))

	// notify
	err = wechat.UpdateToken(&wechatToken, cfg.Corpid, cfg.Corpsecret)
	if err != nil {
		epLog.Error("Notify error. UpdateToken failed. ", err.Error())
		return
	}

	now := time.Now().Format(time.RFC3339)
	log.WithFields(log.Fields{
		"expires_in": wechatToken.Expires_in,
		"next_due":   wechatToken.Next_due,
		"now":        now,
	}).Debug("Update wechatToken")

	content := now + "\nCustom Endpoint HealthCheck:\nNew address for Endpoint " + namespace + "." + epName + "\n" + string(playLoadBytes)
	msg := wechat.WechatMsg{Touser: cfg.Touser, Msgtype: "text", Agentid: cfg.Agentid, Text: map[string]string{"content": content}}
	buf, err := json.Marshal(msg)
	if err != nil {
		epLog.Error("Notify error. json.Marshal(msg) failed: ", err)
		return
	}
	err = wechat.SendMsg(wechatToken.Access_token, buf)
	if err != nil {
		epLog.Error("Notify error. SendMsg failed: ", err.Error())
		return
	} else {
		epLog.Info("Notify succ. To: ", cfg.Touser)
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
func tcpChecker(e corev1.Endpoints, pwg *sync.WaitGroup) {
	epLog := log.WithFields(log.Fields{
		"namespace": e.Namespace,
		"endpoint":  e.Name,
	})

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
		ip := ipaddress{Ipaddress: ips[i], Port: port, Namespace: e.Namespace, Name: e.Name}
		go checkPort(ip, &addresses, &notReadyAddresses, &wg)
	}

	// 等待执行完成
	wg.Wait()

	epLog.Info("Addresses: ", addresses)
	epLog.Info("notReadyAddresses: ", notReadyAddresses)

	addr := epBuilder(addresses, notReadyAddresses, e.Subsets[0].Ports)
	if len(addresses) > 0 {
		if StringSliceEqual(notReadyIps, notReadyAddresses) {
			if len(notReadyAddresses) > 0 {
				epLog.Info("Already Marked notReady IPs. Ignore")
			} else {
				epLog.Info("All endpoints Health. Ignore")
			}
		} else {
			epLog.Debug("notReadyAddresses: ", notReadyAddresses)
			epLog.Debug("notReadyIps: ", notReadyIps)
			// 执行更新前有必要看看线上endpoints是否和 ips 完全一致，防止出现老数据刷掉新数据的情况
			currentEp, err := clientset.CoreV1().Endpoints(e.Namespace).Get(e.Name, metav1.GetOptions{})
			if err != nil {
				return
			}
			currentIPs, _ := getIPs(*currentEp)

			if StringSliceEqual(ips, currentIPs) {
				patchEndpoint(e.Namespace, e.Name, addr)
			} else {
				epLog.Warn("currentIps not same with local ips. Ignore")
				// update local ep
				getEndpoints()
			}
		}
	} else {
		epLog.Warn("No lived ipaddress. Ignore")
	}

	pwg.Done()
}

// do check
func checkPort(ip ipaddress, addresses *[]string, notReadyAddresses *[]string, wg *sync.WaitGroup) {
	epLog := log.WithFields(log.Fields{
		"namespace": ip.Namespace,
		"endpoint":  ip.Name,
	})

	epLog.Debug("Scaning:  ", ip.Ipaddress+":"+ip.Port)

	err := retryPort(ip)

	if err != nil {
		epLog.Debug("notReadyAddresses: ", ip.Ipaddress, " errMsg: ", err.Error())
		mu.Lock()
		*notReadyAddresses = append(*notReadyAddresses, ip.Ipaddress)
		mu.Unlock()
	} else {
		epLog.Debug("Addresses: ", ip.Ipaddress+":"+ip.Port)
		mu.Lock()
		*addresses = append(*addresses, ip.Ipaddress)
		mu.Unlock()
	}

	wg.Done()
}

// retry
func retryPort(ip ipaddress) error {
	var e error
	for i := 0; i < 3; i++ {
		conn, err := net.DialTimeout("tcp", ip.Ipaddress+":"+ip.Port, time.Millisecond*100)
		if conn != nil {
			defer conn.Close()
		}

		if err == nil {
			return err
		} else {
			log.WithFields(log.Fields{
				"namespace": ip.Namespace,
				"endpoint":  ip.Name,
			}).Debug("Dial ", ip.Ipaddress+":"+ip.Port, " failed. will retry: ", i)
			e = err
			time.Sleep(time.Millisecond * 100)
		}
	}
	return e
}

func startedLog() {
	log.WithFields(log.Fields{
		"version":   version,
		"gitHash":   gitHash,
		"buildTime": buildTime,
		"goVersion": goVersion,
	}).Info("ep-healthcheck Started")
}

func appInit() {
	// app config
	err := env.Parse(&cfg)
	if err != nil {
		log.Panic(err.Error())
	}
	listOptions.LabelSelector = cfg.LabelSelector
}

func main() {
	startedLog()
	k8sClientInit()
	appInit()
	logInit()

	// 首先初始化 ep 变量
	getEndpoints()

	var wg sync.WaitGroup
	// 监视 ep 变更事件
	go watchEndpoints()

	for {
		if len(ep) == 0 {
			log.Info("No custom endpoints.")
		}
		for _, e := range ep {
			wg.Add(1)
			// tcp检测
			go tcpChecker(e, &wg)
		}

		wg.Wait()
		time.Sleep(1 * time.Second)
	}
}
