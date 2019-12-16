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

	router.Run("0.0.0.0:9999")
}
