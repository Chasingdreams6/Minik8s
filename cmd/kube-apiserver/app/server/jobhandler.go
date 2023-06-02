package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"minik8s/cmd/kube-apiserver/app/apiconfig"
	"minik8s/pkg/api/core"
	"minik8s/pkg/kube-apiserver/etcd"
	"minik8s/pkg/util/random"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func AddJob(c *gin.Context, s *Server) {
	// TO DO: Add Job
	val, _ := io.ReadAll(c.Request.Body)
	job := core.Job{}
	err := json.Unmarshal([]byte(val), &job)
	if err != nil {
		log.Println(err)
		return
	}
	job.UID = random.GenerateUUID()
	key := c.Request.URL.Path + "/" + job.Name
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
	INFO_HEAD := "[apiserver][getjob]"
	if c.Query("all") == "true" {
		// delete the keys
		res, err := s.Etcdstore.GetWithPrefix(apiconfig.JOB_PATH)
		for i, val := range res {
			var job core.Job
			err = json.Unmarshal([]byte(val.Value), &job)
			if err != nil {
				fmt.Println(INFO_HEAD, err)
				return
			}
			jobStatus := Job2JobStatus(s, &job)
			var data []byte
			data, err = json.Marshal(jobStatus)
			if err != nil {
				fmt.Println(INFO_HEAD, err)
				return
			}
			res[i].Value = string(data)
		}
		if err != nil {
			log.Println(INFO_HEAD, err)
			return
		}
		c.JSON(http.StatusOK, res)
		return
	}
	JobName := c.Query("Name")
	key := c.Request.URL.Path + "/" + string(JobName)

	var res []etcd.ListRes
	var err error

	if c.Query("prefix") == "true" {
		res, err = s.Etcdstore.GetWithPrefix(key)
	} else {
		res, err = s.Etcdstore.GetExact(key)
	}
	// transform job to jobstatus
	if len(res) > 0 {
		var job core.Job
		err = json.Unmarshal([]byte(res[0].Value), &job)
		if err != nil {
			fmt.Println(INFO_HEAD, err)
			return
		}
		fmt.Println(INFO_HEAD, job.Name)
		jobStatus := Job2JobStatus(s, &job)
		var data []byte
		data, err = json.Marshal(jobStatus)
		if err != nil {
			fmt.Println(INFO_HEAD, err)
			return
		}
		res[0].Value = string(data)
		fmt.Println(INFO_HEAD, res[0].Value)
	}
	fmt.Println(INFO_HEAD, res)
	c.JSON(http.StatusOK, res)
}

func DeleteJob(c *gin.Context, s *Server) {
	prefix := "[api-server] [ReplicaSetHandler] [DeleteReplicaSet]"
	fmt.Println(prefix)
	err := c.Request.ParseForm()
	if err != nil {
		return
	}
	if c.Query("all") == "true" {
		// delete the keys
		// delete job file in /home/job
		res, err := s.Etcdstore.GetAll(apiconfig.JOB_PATH)
		for _, val := range res {
			var job core.Job
			err = json.Unmarshal([]byte(val.Value), &job)
			if err != nil {
				fmt.Println(prefix, err)
				return
			}
			dirPath := apiconfig.JOB_FILE_DIR_PATH + "/" + job.Name
			err := os.RemoveAll(dirPath)
			if err != nil {
				fmt.Println(prefix, err)
				return
			}
		}
		num, err := s.Etcdstore.DelAll(apiconfig.JOB_PATH)
		if err != nil {
			log.Println(err)
			return
		}
		num, err = s.Etcdstore.DelAll(apiconfig.JOB_FILE_PATH)
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

	Name := c.Query("JobName")
	fmt.Println("JobName:", Name)
	key := c.Request.URL.Path + "/" + Name
	err = s.Etcdstore.Del(key)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "delete job failed",
			"error":   err,
		})
		return
	}
	key = apiconfig.JOB_FILE_PATH + "/" + Name
	err = s.Etcdstore.Del(key)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "delete job failed",
			"error":   err,
		})
		return
	}
	if Name != "" {
		os.RemoveAll(apiconfig.JOB_FILE_DIR_PATH + "/" + Name)
	}
	c.JSON(http.StatusOK, gin.H{
		"message":       "delete job success",
		"deletePodName": Name,
	})
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

func Job2JobStatus(s *Server, job *core.Job) core.JobStatus {
	jobStatus := core.JobStatus{}
	jobStatus.JobName = job.Name
	jobStatus.Status = string(core.JobPending)
	res, err := s.Etcdstore.GetExact(apiconfig.POD_PATH + "/" + job.Name)
	if err != nil || len(res) == 0 {
		fmt.Println("[apiserver][getjob] get pod failed")
		return jobStatus
	}
	var pod core.Pod
	err = json.Unmarshal([]byte(res[0].Value), &pod)
	if pod.Status.Phase == core.PodSucceeded {
		jobStatus.Status = string(core.JobSuccess)
	} else if pod.Status.Phase == core.PodFailed {
		jobStatus.Status = string(core.JobFailed)
	}
	return jobStatus
}
