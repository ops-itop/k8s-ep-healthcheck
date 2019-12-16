package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type Foo struct {
	Value1 int
	Value2 string
}

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

func main() {
	router := gin.Default()
	router.LoadHTMLGlob("templates/*")

	router.GET("/array", func(c *gin.Context) {
		var values []int
		for i := 0; i < 5; i++ {
			values = append(values, i)
		}

		c.HTML(http.StatusOK, "array.tmpl", gin.H{"values": values})
	})

	router.GET("/struct", func(c *gin.Context) {
		var values []Foo
		for i := 0; i < 5; i++ {
			values = append(values, Foo{Value1: i, Value2: "value " + strconv.Itoa(i)})
		}

		c.HTML(http.StatusOK, "struct.tmpl", gin.H{"values": values})
	})

	router.GET("/map", func(c *gin.Context) {
		values := make(map[string]string)
		values["language"] = "Go"
		values["version"] = "1.7.4"

		c.HTML(http.StatusOK, "map.tmpl", gin.H{"myMap": values})
	})

	router.GET("/mapkeys", func(c *gin.Context) {
		values := make(map[string]string)
		values["language"] = "Go"
		values["version"] = "1.7.4"

		c.HTML(http.StatusOK, "mapkeys.tmpl", gin.H{"myMap": values})
	})

	router.GET("/mapSelectKeys", func(c *gin.Context) {
		values := make(map[string]string)
		values["language"] = "Go"
		values["version"] = "1.7.4"

		c.HTML(http.StatusOK, "mapSelectKeys.tmpl", gin.H{"myMap": values})
	})

	stat := Stat{
		Unhealth: map[string]StatEp{
			"dev.ep1": StatEp{
				Name:      "ep1",
				Namespace: "dev",
				Status:    0,
				Port:      "80",
				Addresses: map[string]StatAddr{
					"10.0.0.1": StatAddr{
						Ip:     "10.0.0.1",
						Status: 0,
						Succ:   22,
						Failed: 33,
					},
				},
			},
		},
	}

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"Unhealth": stat.Unhealth,
			"Health":   stat.Health,
		})
	})
	router.GET("/stat", func(c *gin.Context) {
		c.JSON(http.StatusOK, stat)
	})

	router.Run("0.0.0.0:9999")
}
