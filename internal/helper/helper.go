package helper

import (
	corev1 "k8s.io/api/core/v1"
)

// convert ip list to endpoints addresses list
func AddrBuilder(addrs []string) []interface{} {
	addrList := make([]interface{}, 0)

	for _, v := range addrs {
		item := map[string]string{"ip": v}
		addrList = append(addrList, item)
	}

	return addrList
}

// build new endpoints subsets
func EndpointBuilder(addresses []string, notReadyAddresses []string, ports []corev1.EndpointPort) map[string]interface{} {
	addr := make(map[string]interface{})
	subsets := make([]interface{}, 0)
	item := make(map[string]interface{})

	item["notReadyAddresses"] = AddrBuilder(notReadyAddresses)
	item["addresses"] = AddrBuilder(addresses)
	item["ports"] = ports

	subsets = append(subsets, item)
	addr["subsets"] = subsets

	return addr
}

func GetAddresses(e corev1.Endpoints) ([]string, []string) {
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
