package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"minik8s/cmd/kube-apiserver/app/apiconfig"
	"minik8s/pkg/api/core"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AddJob(c *gin.Context, s *Server) {
	// TO DO: Add Job
	fmt.Println("in add job")
	val, _ := io.ReadAll(c.Request.Body)
	job := core.Job{}
	err := json.Unmarshal([]byte(val), &job)
	if err != nil {
		log.Println(err)
		return
	}
	key := c.Request.URL.Path + "/" + job.Name
	fmt.Println(key)
	res, _ := s.Etcdstore.Get(key)
	if len(res) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Job Name Duplicate.",
		})
		return
	}
	// node data
	err = s.Etcdstore.Put(key, string(val))
	if err != nil {
		fmt.Print(err)
		return
	}
}

func GetJob(c *gin.Context, s *Server) {
	if c.Query("all") == "true" {
		// delete the keys
		res, err := s.Etcdstore.GetWithPrefix(apiconfig.JOB_PATH)
		if err != nil {
			log.Println(err)
			return
		}
		c.JSON(http.StatusOK, res)
		return
	}
	JobName := c.Query("Job")
	key := c.Request.URL.Path + "/" + string(JobName)
	res, err := s.Etcdstore.Get(key)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "etcd get job failed",
		})
		return
	}
	c.JSON(http.StatusOK, res[0])
}

func AddJobFile(c *gin.Context, s *Server) {
	val, _ := io.ReadAll(c.Request.Body)
	job := core.JobUpload{}
	err := json.Unmarshal([]byte(val), &job)
	if err != nil {
		log.Println(err)
		return
	}
	key := c.Request.URL.Path + "/" + job.JobName
	res, _ := s.Etcdstore.Get(key)
	if len(res) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Job Name Duplicate.",
		})
		return
	}
	// node data
	err = s.Etcdstore.Put(key, string(val))
	if err != nil {
		fmt.Print(err)
		return
	}
}

func GetJobFile(c *gin.Context, s *Server) {
	JobName := c.Query("JobName")
	fmt.Println("[apiserver][getjobfile] JobName", JobName)
	key := c.Request.URL.Path + "/" + string(JobName)
	res, err := s.Etcdstore.Get(key)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "etcd get node failed",
		})
		return
	}
	job := core.JobUpload{}
	err = json.Unmarshal([]byte(res[0]), &job)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("[apiserver][getjobfile] res", string(job.Program))
	c.JSON(http.StatusOK, res[0])
}
