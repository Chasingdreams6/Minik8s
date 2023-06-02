package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"minik8s/cmd/kube-apiserver/app/apiconfig"
	"minik8s/pkg/api/core"
	v1 "minik8s/pkg/apis/meta/v1"
	"minik8s/pkg/kube-apiserver/etcd"
	"minik8s/pkg/util/random"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AddWorkflow(c *gin.Context, s *Server) {
	prefix := "[api-server] [AddWorkflow] "
	val, _ := io.ReadAll(c.Request.Body)
	w := core.Workflow{}
	err := json.Unmarshal([]byte(val), &w)
	if err != nil {
		log.Println("[ERROR] ", prefix, err)
		return
	}
	key := c.Request.URL.Path + "/" + w.Name
	ServerGetFunc := func(name string) (core.Function, error) {
		return GetFunc(s, name)
	}
	dag, err := w.Workflow2DAG(ServerGetFunc)
	if err != nil {
		log.Println("[ERROR] ", prefix, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}
	dag.Name = w.Name
	dag.Input = w.Spec.Input
	dag.UID = random.GenerateUUID()
	dag.ObjectMeta.CreationTimestamp = v1.Now()
	dag.Result = "Pending"
	fmt.Println("[Node]:", dag.Nodes)
	fmt.Println("[Edge]:", dag.Edges)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err,
		})
	}

	body, _ := json.Marshal(dag)

	err = s.Etcdstore.Put(key, string(body))
	if err != nil {
		log.Println("[ERROR] ", prefix, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Add Workflow Success.",
	})
}

// GetWorkflow Body传入Name
func GetWorkflow(c *gin.Context, s *Server) {
	prefix := "[api-server] [WorkflowHandler] [GetWorkflow]"
	fmt.Println(prefix)
	if c.Query("all") == "true" {
		// delete the keys
		res, err := s.Etcdstore.GetWithPrefix(apiconfig.WORKFLOW_PATH)
		if err != nil {
			log.Println(err)
			return
		}
		c.JSON(http.StatusOK, res)
		return
	}

	Name := c.Query("Name")
	key := c.Request.URL.Path + "/" + string(Name)

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
			"message": "etcd get Workflow failed",
		})
		return
	}

	c.JSON(http.StatusOK, res)
}

func DeleteWorkflow(c *gin.Context, s *Server) {
	prefix := "[api-server] [WorkflowHandler] [DeleteWorkflow]"
	fmt.Println(prefix)
	err := c.Request.ParseForm()
	if err != nil {
		return
	}
	if c.Query("all") == "true" {
		// delete the keys
		num, err := s.Etcdstore.DelAll(apiconfig.WORKFLOW_PATH)
		if err != nil {
			log.Println(err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":   "delete all pods successfully.",
			"deleteNum": num,
		})
		return
	}

	Name := c.Query("Name")
	fmt.Println("Name:", Name)
	key := c.Request.URL.Path + "/" + Name
	err = s.Etcdstore.Del(key)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "delete Workflow failed",
			"error":   err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":       "delete Workflow success",
		"deletePodName": Name,
	})
}

func UpdateWorkflow(c *gin.Context, s *Server) {
	val, _ := io.ReadAll(c.Request.Body)
	w := core.DAG{}
	err := json.Unmarshal([]byte(val), &w)
	if err != nil {
		log.Println(err)
		return
	}
	key := c.Request.URL.Path + "/" + w.Name
	body, _ := json.Marshal(w)
	err = s.Etcdstore.Put(key, string(body))
	if err != nil {
		log.Println(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Update Workflow Success.",
	})
}

func GetFunc(s *Server, resource string) (core.Function, error) {
	prefix := "[api-server] [WorkflowHandler] [GetFunc]"
	fmt.Println(prefix)
	key := apiconfig.FUNCTION_PATH + "/" + resource
	res, err := s.Etcdstore.GetExact(key)
	if err != nil {
		log.Println(err)
		return core.Function{}, err
	}
	if len(res) == 0 {
		return core.Function{}, fmt.Errorf("function %s not found", resource)
	}
	f := core.Function{}
	err = json.Unmarshal([]byte(res[0].Value), &f)
	if err != nil {
		log.Println(err)
		return core.Function{}, err
	}
	return f, nil
}
