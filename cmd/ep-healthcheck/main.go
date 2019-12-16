package main

import (
	"encoding/json"
	//"reflect"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	//"k8s.io/apimachinery/pkg/api/errors"
	"github.com/ops-itop/k8s-ep-healthcheck/internal/config"
	"github.com/ops-itop/k8s-ep-healthcheck/internal/helper"
	"github.com/ops-itop/k8s-ep-healthcheck/internal/stat"
	"github.com/ops-itop/k8s-ep-healthcheck/pkg/notify/wechat"
	"github.com/ops-itop/k8s-ep-healthcheck/pkg/utils"
	log "github.com/sirupsen/logrus"
	tcp "github.com/tevino/tcp-shaker"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

// version info
var (
	gitHash   string
	version   string
	buildTime string
	goVersion string
)

// one ipaddress for scaning
type ipaddress struct {
	Namespace string
	Name      string
	Ipaddress string
	Port      string
}

var (
	clientset   *kubernetes.Clientset
	cfg         config.Config
	mu          sync.RWMutex
	ep          []corev1.Endpoints // store all endpoints
	wechatToken wechat.AccessToken
	listOptions metav1.ListOptions //labelSelector for endpoints

	st stat.Stat
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
	defer mu.Unlock()
	ep = endpoints.Items
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

// tcp checker
func tcpChecker(e corev1.Endpoints, pwg *sync.WaitGroup) {
	defer pwg.Done()
	epLog := log.WithFields(log.Fields{
		"namespace": e.Namespace,
		"endpoint":  e.Name,
	})

	ips, notReadyIps := helper.GetAddresses(e)
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
	epLog.Warn("notReadyAddresses: ", notReadyAddresses)

	// Statistics
	st.Update(e.Namespace, e.Name, addresses, notReadyAddresses, port)

	addr := helper.EndpointBuilder(addresses, notReadyAddresses, e.Subsets[0].Ports)
	if len(addresses) > 0 {
		if utils.StringSliceEqual(notReadyIps, notReadyAddresses) {
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
				epLog.Error("get currentEp error: ", err.Error())
				return
			}
			currentIPs, _ := helper.GetAddresses(*currentEp)

			if utils.StringSliceEqual(ips, currentIPs) {
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
}

// do check
func checkPort(ip ipaddress, addresses *[]string, notReadyAddresses *[]string, wg *sync.WaitGroup) {
	defer wg.Done()
	epLog := log.WithFields(log.Fields{
		"namespace": ip.Namespace,
		"endpoint":  ip.Name,
	})

	epLog.Trace("Scaning:  ", ip.Ipaddress+":"+ip.Port)

	err := retryPort(ip)

	switch err {
	case tcp.ErrTimeout:
		epLog.Warn("Tcp check error: ", ip.Ipaddress, " errMsg: ", err.Error())
		mu.Lock()
		defer mu.Unlock()
		*notReadyAddresses = append(*notReadyAddresses, ip.Ipaddress)
	case nil:
		epLog.Trace("Tcp check succeeded: ", ip.Ipaddress+":"+ip.Port)
		mu.Lock()
		defer mu.Unlock()
		*addresses = append(*addresses, ip.Ipaddress)
	default:
		epLog.Error("Error occurred while connecting: ", ip.Ipaddress+":"+ip.Port, " errMsg: ", err)
	}
}

// retry
func retryPort(ip ipaddress) error {
	var e error
	c := tcp.NewChecker()

	ctx, stopChecker := context.WithCancel(context.Background())
	defer stopChecker()
	go func() {
		if err := c.CheckingLoop(ctx); err != nil {
			log.Error("checking loop stopped due to fatal error: ", err)
		}
	}()

	<-c.WaitReady()

	for i := 0; i < cfg.Retry; i++ {
		err := c.CheckAddr(ip.Ipaddress+":"+ip.Port, time.Millisecond*time.Duration(cfg.Timeout))

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
	err := cfg.Init()
	if err != nil {
		log.Panic(err.Error())
	}
	listOptions.LabelSelector = cfg.LabelSelector

	st.Init()
}

func doCheck() {
	if len(ep) == 0 {
		log.Info("No custom endpoints.")
	}
	var wg sync.WaitGroup
	for _, e := range ep {
		wg.Add(1)
		// tcp检测
		go tcpChecker(e, &wg)
	}
	wg.Wait()
}

func api(r *gin.Engine) {
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.tmpl", gin.H{
			"Unhealth": st.Unhealth,
			"Health":   st.Health,
		})
	})

	r.GET("/endpoints", func(c *gin.Context) {
		c.JSON(200, ep)
	})

	r.GET("/stat", func(c *gin.Context) {
		c.JSON(200, st)
	})
}

func main() {
	startedLog()
	k8sClientInit()
	appInit()
	logInit()

	// 首先初始化 ep 变量
	getEndpoints()

	// 监视 ep 变更事件
	go watchEndpoints()

	router := gin.Default()
	pprof.Register(router)
	api(router)
	go func() {
		router.Run()
	}()

	for {
		doCheck()
		time.Sleep(time.Duration(cfg.Interval) * time.Second)
	}
}
