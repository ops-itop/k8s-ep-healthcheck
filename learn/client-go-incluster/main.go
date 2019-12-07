package main

import (
	"encoding/json"
	"log"
	//"reflect"
	"time"

	//"k8s.io/apimachinery/pkg/api/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// global var store all endpoints
var ep []corev1.Endpoints

func _init(c *kubernetes.Clientset, l metav1.ListOptions) {
	endpoints, err := c.CoreV1().Endpoints("").List(l)
	if err != nil {
		panic(err.Error())
	}
	ep = endpoints.Items
	//epStr, _ := json.MarshalIndent(ep, "", " ")
	//log.Printf("Endpionts: %s\n", epStr)
}

// patch endpoint
func update(c *kubernetes.Clientset, namespace string, epName string, data map[string]interface{}) {
	playLoadBytes, _ := json.Marshal(data)

	_, err := c.CoreV1().Endpoints(namespace).Patch(epName, types.StrategicMergePatchType, playLoadBytes)

	if err != nil {
		log.Printf("Update Ednpoint %v Error: %v", epName, err)
	}
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

	// init
	_init(clientset, listOptions)
	for {
		for _, e := range ep {
			log.Println(e.Subsets[0])
			addr := map[string]interface{}{
				"subsets": []interface{}{
					0: map[string]interface{}{
						"notReadyAddresses": []interface{}{map[string]string{
							"ip": "39.156.69.79"}},
						"addresses": e.Subsets[0].Addresses,
						"ports":     e.Subsets[0].Ports}}}
			update(clientset, e.Namespace, e.Name, addr)

			addrStr, _ := json.Marshal(addr)
			log.Printf("New addresses for Endpoint %v: %v", e.Name, string(addrStr))
		}
		time.Sleep(1 * time.Second)
	}
}
