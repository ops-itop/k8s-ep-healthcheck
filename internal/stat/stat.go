package stat

import (
	"github.com/ops-itop/k8s-ep-healthcheck/pkg/utils"
	"sync"
)

var mu sync.RWMutex

const MAXCOUNT = 2592000 // 计数器最大值，超过清零

// Statistics for health check
type Stat struct {
	Unhealth map[string]StatEp `json:"unhealth"`
	Health   map[string]StatEp `json:"health"`
}

type StatEp struct {
	Name      string              `json:"name"`
	Namespace string              `json:"namespace"`
	Status    int                 `json:"status"`
	Addresses map[string]StatAddr `json:"addresses"`
	Port      string              `json:"port"`
}

type StatAddr struct {
	Ip     string `json:"ip"`
	Status int    `json:"status"`
	Succ   int    `json:"succ"`
	Failed int    `json""failed"`
}

func (st *Stat) Init() {
	st.Unhealth = make(map[string]StatEp)
	st.Health = make(map[string]StatEp)
}

func (ep *StatEp) Init(namespace string, name string, status int, port string) {
	ep.Name = name
	ep.Namespace = namespace
	ep.Status = status
	ep.Addresses = make(map[string]StatAddr)
	ep.Port = port
}

func (addr *StatAddr) Init(ip string, status int) {
	addr.Ip = ip
	addr.Status = status
	addr.Succ = 0
	addr.Failed = 0
}

func (ep *StatEp) Update() {
}

func (addr *StatAddr) Update(ip string, status int, succ int, failed int) {
	addr.Ip = ip
	addr.Status = status
	addr.Succ += succ
	addr.Failed += failed

	if addr.Succ > MAXCOUNT {
		addr.Succ = 0
	}

	if addr.Failed > MAXCOUNT {
		addr.Failed = 0
	}
}

func remove(m map[string]StatEp, n map[string]StatEp, key string) {
	if _, ok := m[key]; ok {
		n[key] = m[key]
		delete(m, key)
	}
}

func update(m map[string]StatEp, status int, namespace string, name string, addresses []string, notReadyAddresses []string, port string) {
	key := namespace + "." + name
	ips := append(addresses, notReadyAddresses...)

	endpoint := m[key]

	if _, ok := m[key]; !ok {
		endpoint.Init(namespace, name, status, port)
	} else {
		for _, v := range ips {
			ipaddr := endpoint.Addresses[v]
			ipstat := utils.BoolToInt(utils.Contains(addresses, v))
			if _, ok := m[key].Addresses[v]; !ok {
				ipaddr.Init(v, ipstat)
			} else {
				succ := utils.BoolToInt(utils.Contains(addresses, v))
				failed := utils.BoolToInt(utils.Contains(notReadyAddresses, v))
				ipaddr.Update(v, ipstat, succ, failed)
			}

			m[key].Addresses[v] = ipaddr
		}
	}

	m[key] = endpoint
}

// update Statistics
func (st *Stat) Update(namespace string, name string, addresses []string, notReadyAddresses []string, port string) {
	mu.Lock()
	defer mu.Unlock()

	key := namespace + "." + name
	status := 1
	count := len(notReadyAddresses)

	if count == 0 {
		remove(st.Unhealth, st.Health, key)
		update(st.Health, status, namespace, name, addresses, notReadyAddresses, port)
	} else {
		status = 0
		remove(st.Health, st.Unhealth, key)
		update(st.Unhealth, status, namespace, name, addresses, notReadyAddresses, port)
	}
}
