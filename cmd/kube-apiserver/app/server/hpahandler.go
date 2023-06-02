package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"minik8s/cmd/kube-apiserver/app/apiconfig"
	"minik8s/pkg/api/core"
	"minik8s/pkg/kube-apiserver/etcd"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetHPA(c *gin.Context, s *Server) {
	prefix := "[api-server] [functionHandler] [GetHPA]"
	fmt.Println(prefix)
	if c.Query("all") == "true" {
		// delete the keys
		res, err := s.Etcdstore.GetWithPrefix(apiconfig.HPA_PATH)
		if err != nil {
			log.Println(err)
			return
		}
		c.JSON(http.StatusOK, res)
		return
	}

	HPAName := c.Query("Name")
	key := c.Request.URL.Path + "/" + string(HPAName)

	var res []etcd.ListRes
	var err error

	if c.Query("prefix") == "true" {
		res, err = s.Etcdstore.GetWithPrefix(key)
		fmt.Println(res)
	} else {
		res, err = s.Etcdstore.GetExact(key)
	}

	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "etcd get HPA failed",
		})
		return
	}

	c.JSON(http.StatusOK, res)

}

func AddHPA(c *gin.Context, s *Server) {
	val, _ := io.ReadAll(c.Request.Body)
	hpa := core.HPA{}
	err := json.Unmarshal([]byte(val), &hpa)
	if err != nil {
		log.Println(err)
		return
	}
	key := c.Request.URL.Path + "/" + hpa.Name
	res, _ := s.Etcdstore.Get(key)
	if len(res) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "HPA Name Duplicate.",
		})
		return
	}
	err = s.Etcdstore.Put(key, string(val))
	if err != nil {
		fmt.Print(err)
		return
	}
}

func DeleteHPA(c *gin.Context, s *Server) {

	if c.Query("all") == "true" {
		// delete the keys
		_, err := s.Etcdstore.DelAll(apiconfig.HPA_PATH)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "etcd delete HPA failed",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "delete HPAs success",
		})
		return
	}

	HPAName := c.Query("HPAName")
	key := c.Request.URL.Path + "/" + string(HPAName)
	err := s.Etcdstore.Del(key)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "etcd delete HPA failed",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "delete HPA success",
	})
}
