package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"minik8s/cmd/kube-apiserver/app/apiconfig"
	"minik8s/pkg/api/core"
	v1 "minik8s/pkg/apis/meta/v1"
	"minik8s/pkg/cmd/create"
	"minik8s/pkg/kube-apiserver/etcd"
	"minik8s/pkg/util/random"
	"net/http"
	"strings"
	"time"
)

func GetFunction(c *gin.Context, s *Server) {
	prefix := "[api-server] [functionHandler] [GetFunction]"
	fmt.Println(prefix)
	if c.Query("all") == "true" {
		// delete the keys
		res, err := s.Etcdstore.GetWithPrefix(apiconfig.FUNCTION_PATH)
		if err != nil {
			log.Println(err)
			return
		}
		c.JSON(http.StatusOK, res)
		return
	}

	FunctionName := c.Query("Name")
	key := c.Request.URL.Path + "/" + string(FunctionName)

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
			"message": "etcd get funtion failed",
		})
		return
	}

	c.JSON(http.StatusOK, res)

}

func DeleteFunction(c *gin.Context, s *Server) {
	prefix := "[api-server] [functionHandler] [DeleteFunction]"
	fmt.Println(prefix)
	err := c.Request.ParseForm()
	if err != nil {
		return
	}
	if c.Query("all") == "true" {
		// delete the keys
		num, err := s.Etcdstore.DelAll(apiconfig.FUNCTION_PATH)
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

	FunctionName := c.Query("FunctionName")
	fmt.Println("FunctionName:", FunctionName)
	key := c.Request.URL.Path + "/" + FunctionName
	err = s.Etcdstore.Del(key)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "delete Function failed",
			"error":   err,
		})
		return
	}

	deletePodWithFunctionName(s, FunctionName)

	c.JSON(http.StatusOK, gin.H{
		"message":            "delete Function success",
		"deleteFunctionName": FunctionName,
	})
}

func AddFunction(c *gin.Context, s *Server) {
	prefix := "[api-server] [AddFunction] "
	val, _ := io.ReadAll(c.Request.Body)
	function := core.Function{}
	err := json.Unmarshal([]byte(val), &function)
	if err != nil {
		log.Println("[ERROR] ", prefix, err)
		return
	}
	key := c.Request.URL.Path + "/" + function.Name
	res, _ := s.Etcdstore.Get(key)
	if len(res) > 0 {
		// 删除所有之前这个Function版本的Pod
		deletePodWithFunctionName(s, function.Name)
	}

	function.UID = random.GenerateUUID()
	function.ObjectMeta.CreationTimestamp = v1.Now()

	body, _ := json.Marshal(function)

	err = s.Etcdstore.Put(key, string(body))
	if err != nil {
		log.Println("[ERROR] ", prefix, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "add function success.",
	})
}

func deletePodWithFunctionName(s *Server, func_name string) {
	res, err := s.Etcdstore.GetWithPrefix(apiconfig.POD_PATH)
	if err != nil {
		fmt.Println("[ERROR] [deletePodWithFunctionName] ", err)
		return
	}
	for _, v := range res {
		pod := core.Pod{}
		_ = json.Unmarshal([]byte(v.Value), &pod)
		if len(pod.OwnerReferences) > 0 && pod.OwnerReferences[0].Name == func_name && pod.OwnerReferences[0].Kind == "Function" {
			_ = s.Etcdstore.Del(v.Key)
		}
	}
}

func scheduler(s *Server, func_name string) (*core.Pod, error) {
	var p []*core.Pod = make([]*core.Pod, 0)
	res, err := s.Etcdstore.GetWithPrefix(apiconfig.POD_PATH)
	if err != nil {
		fmt.Println("[ERROR] [scheduler] ", err)
		return nil, err
	}
	for _, v := range res {
		pod := &core.Pod{}
		_ = json.Unmarshal([]byte(v.Value), pod)
		if len(pod.OwnerReferences) > 0 && pod.OwnerReferences[0].Name == func_name && pod.OwnerReferences[0].Kind == "Function" {
			p = append(p, pod)
		}
	}

	if len(p) == 0 {
		fmt.Println("[ERROR] [scheduler] no pod instance")
		return nil, nil
	}

	function := core.Function{}
	r, err := s.Etcdstore.GetExact(apiconfig.FUNCTION_PATH + "/" + func_name)
	if err != nil {
		fmt.Println("[ERROR] [scheduler] [GetExact]", err)
		return nil, err
	}
	err = json.Unmarshal([]byte(r[0].Value), &function)
	if err != nil {
		fmt.Println("[ERROR] [scheduler] [Unmarshal]", err)
		return nil, err
	}
	function.Spec.InvokeTimes++
	data, err := json.Marshal(function)
	if err != nil {
		fmt.Println("[ERROR] [scheduler] [Marshal]", err)
		return nil, err
	}
	err = s.Etcdstore.Put(apiconfig.FUNCTION_PATH+"/"+func_name, string(data))
	if err != nil {
		fmt.Println("[ERROR] [scheduler] [Put]", err)
		return nil, err
	}
	return p[function.Spec.InvokeTimes%len(p)], nil
}

func GetFunctionPod(function_name string, image_name string) *core.Pod {
	pod_name := "function-" + function_name + "-" + random.GenerateRandomString(5)
	return &core.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name: pod_name,
			OwnerReferences: []v1.OwnerReference{
				{
					Kind: "Function",
					Name: function_name,
				},
			},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:  "container",
					Image: image_name,
				},
			},
		},
	}
}

func InvokeFunction(c *gin.Context, s *Server) {
	function_name := c.Param("function_name")[1:]

	x, err := s.Etcdstore.GetExact(apiconfig.FUNCTION_PATH + "/" + function_name)
	if err != nil {
		fmt.Println("[InvokeFunction] ", "Error getting function:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error getting function.",
		})
		return
	}
	if len(x) == 0 {
		fmt.Println("[InvokeFunction] ", "Function not found.")
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Function not found.",
		})
		return
	}

	function := core.Function{}
	res, err := s.Etcdstore.GetExact(apiconfig.FUNCTION_PATH + "/" + function_name)
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(res[0].Value), &function)

	pod, err := scheduler(s, function_name)
	var podIP string

	if pod == nil {
		pod = GetFunctionPod(function_name, function.Spec.Image)

		err = create.CreatePod(pod)
		if err != nil {
			fmt.Println("[InvokeFunction] ", "Error creating pod:", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Error creating pod.",
			})
			return
		}

		for {
			r, err := s.Etcdstore.GetExact(apiconfig.POD_PATH + "/" + pod.Name)
			if err != nil {
				fmt.Println("[InvokeFunction] ", "Error getting pod:", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Error getting pod.",
				})
				return
			}
			if len(r) > 0 {
				if strings.Contains(r[0].Value, `"podIP":"`) {
					err = json.Unmarshal([]byte(r[0].Value), pod)
					if err != nil {
						fmt.Println("[InvokeFunction] ", "Error unmarshalling pod:", err)
						c.JSON(http.StatusInternalServerError, gin.H{
							"message": "Error unmarshalling pod.",
						})
						return
					}
					if pod.Status.PodIP != "" {
						podIP = pod.Status.PodIP
						time.Sleep(5 * time.Second)
						break
					}
				}
			}
			time.Sleep(1 * time.Second)
		}
	} else {
		podIP = pod.Status.PodIP
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return
	}
	fmt.Println("[InvokeFunction] ", "PodIP:", podIP, "body:", string(body))
	req, err := http.NewRequest("POST", "http://"+podIP+":8888/function/my_module/"+function_name, bytes.NewReader(body))
	if err != nil {
		fmt.Println("[InvokeFunction] ", "Error creating request:", err)
		return
	}

	// 设置请求头的 Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("[InvokeFunction]", "Error sending request:", err)
		return
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	fmt.Println("[InvokeFunction] ", "respond:", string(body))

	// 设置响应头
	c.Header("Content-Type", resp.Header.Get("Content-Type"))

	// 返回响应体
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}

func ScaleFunction(c *gin.Context, s *Server) {
	prefix := "[api-server] [ScaleFunction] "

	cbody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return
	}

	type ScaleBody struct {
		Message       string `json:"message" yaml:"message"`
		Function_name string `json:"function_name" yaml:"function_name"`
	}

	var body ScaleBody

	err = json.Unmarshal(cbody, &body)
	if err != nil {
		fmt.Println(prefix, "Error unmarshalling body:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error unmarshalling body.",
		})
		return
	}

	fmt.Println(prefix, "function name:", body.Function_name)

	// get function from etcd
	x, err := s.Etcdstore.GetExact(apiconfig.FUNCTION_PATH + "/" + body.Function_name)
	if err != nil {
		fmt.Println(prefix, "Error getting function:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error getting function.",
		})
		return
	}
	if len(x) == 0 {
		fmt.Println(prefix, "Function not found.")
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Function not found.",
		})
		return
	}
	function := core.Function{}
	err = json.Unmarshal([]byte(x[0].Value), &function)
	if err != nil {
		fmt.Println(prefix, "Error unmarshalling function:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error unmarshalling function.",
		})
		return
	}

	pod := GetFunctionPod(body.Function_name, function.Spec.Image)

	err = create.CreatePod(pod)
	if err != nil {
		fmt.Println(prefix, "Error creating pod:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error creating pod.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK.",
	})
}
