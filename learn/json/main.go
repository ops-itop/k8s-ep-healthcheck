package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	v := map[string]interface{}{"key": []interface{}{map[string]interface{}{"addr": []interface{}{map[string]string{"ip": "1.0.0.2"}}}}}

	vStr, _ := json.Marshal(v)
	fmt.Println(string(vStr))

	//addr := map[string]interface{}{"subsets": []interface{}{"addresses": []interface{}{map[string]string{"ip": "39.156.69.79"}}}}
	//addr := map[string]interface{}{"subsets": []interface{}{"addresses": []interface{}{map[string]string{"ip": "39.156.69.79"}}}}
	//addrStr, _ := json.Marshal(addr)
	//fmt.Println(string(addrStr))
}
