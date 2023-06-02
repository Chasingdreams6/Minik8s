package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"minik8s/cmd/kube-apiserver/app/apiconfig"
	"minik8s/pkg/api/core"
	"minik8s/pkg/service"
	myJson "minik8s/pkg/util/json"
	"minik8s/pkg/util/log"
	"net/http"
	"strings"
	"time"
)

type EventType string

const (
	Added    EventType = "ADDED"
	Modified EventType = "MODIFIED"
	Deleted  EventType = "DELETED"
	Error    EventType = "ERROR"
)

type Event struct {
	Type EventType
	Key  string
	Val  string
}

type ListRes struct {
	ResourceVersion int64
	Key             string
	Value           string
}

func List(resource string) []ListRes {
	url := apiconfig.Server_URL + resource + "?all=true"
	for {
		resp, err := http.Get(url)
		if err != nil {
			// handle error
			fmt.Println("[httpclient] [List] web get error:", err)
			time.Sleep(1 * time.Second)
			continue
		}
		defer resp.Body.Close()
		reader := resp.Body
		data, err := io.ReadAll(reader)
		if err != nil {
			fmt.Println("[httpclient] [List] read error:", err)
			time.Sleep(1 * time.Second)
			continue
		}
		var resList []ListRes
		err = json.Unmarshal(data, &resList)
		if err != nil {
			return nil
		}
		fmt.Println("[httpclient] [List] ", resList)
		return resList
	}
}

func Watch(resourses string) WatchInterface {
	watcher := &watcher{}
	watcher.resultChan = make(chan Event, 1000)
	reader := func(wc chan<- Event) {
		for {
			fmt.Println("[httpclient] [Watch] start watch")
			url := apiconfig.Server_URL + "/watch" + resourses + "?prefix=true"
			resp, err := http.Get(url)
			buf := make([]byte, 40960)
			if err != nil {
				// handle error
				fmt.Println("[httpclient] [Watch] web get error:", err)
				goto Reconnect
			}
			defer resp.Body.Close()
			// Need to optimize
			for {
				n, err := resp.Body.Read(buf)
				if n != 0 {
					event := Event{}
					strings := myJson.ExtractNestedContent(string(buf[:n]))
					for _, s := range strings {
						if len(s) > 0 {
							err = json.Unmarshal([]byte(s), &event)
							if err != nil {
								fmt.Println("[httpclient] [Watch] unmarshal:", err)
								continue
							}
							fmt.Println("[httpclient] [Watch] unmarshal:", event.Key, event.Val, event.Type)
							// send event to watcher.resultChan
							wc <- event
						}
					}
				} else {
					fmt.Println("[httpclient] [Watch] break", err)
					goto Reconnect
				}
				time.Sleep(1 * time.Second)
			}
			//todo : try to reconnect
		Reconnect:
			fmt.Println("[httpclient] [Watch] try to reconnect")
			time.Sleep(1 * time.Second)
		}
	}
	go reader(watcher.resultChan)
	return watcher
}

func AddPod(pod *core.Pod) error {
	url := apiconfig.Server_URL + apiconfig.POD_PATH
	data, err := json.Marshal(pod)
	if err != nil {
		fmt.Println("failed to marshal person:", err)
		return err
	}
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	fmt.Println("[http][AddPod]", string(data))
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		fmt.Println("add pod success")
	} else {
		return fmt.Errorf("add pod failed")
	}
	return nil
}

func UpdatePod(pod *core.Pod) error {
	url := apiconfig.Server_URL + apiconfig.POD_PATH
	data, err := json.Marshal(pod)
	if err != nil {
		fmt.Println("[httpclient] [UpdatePod] failed to marshal person:", err)
		return err
	} else {
		fmt.Println("[httpclient] [UpdatePod] update pod:", string(data))
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		fmt.Println("[httpclient] [UpdatePod] Error:", err)
		return err
	}
	defer resp.Body.Close()
	fmt.Println("[httpclient] [UpdatePod] Response Status:", resp.Status)

	return nil
}

func DeletePod(PodName string) error {
	url := apiconfig.Server_URL + apiconfig.POD_PATH + "?" + "PodName=" + PodName
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	err = log.CheckHttpStatus("[httpclient] [DeletePod] ", resp)
	return nil
}

func AddNode(node *core.Node) error {
	url := apiconfig.Server_URL + apiconfig.NODE_PATH
	fmt.Println("[http]: ", url)
	data, err := json.Marshal(node)
	if err != nil {
		return err
	}
	for {
		req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		// 发送请求
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		defer resp.Body.Close()
		err = log.CheckHttpStatus("[httpclient] [AddNode] ", resp)
		if err != nil {
			return err
		}
		return nil
	}
}
func GetService(name string) (*service.Service, error) {
	prefix := "[tool][GetService]"
	fmt.Println(prefix + "key:" + name)
	url := apiconfig.Server_URL + apiconfig.SERVICE_PATH + "?" + "Name=" + name
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf := make([]byte, 40960)
	var res []ListRes
	n, err := resp.Body.Read(buf)
	if n != 0 || err != io.EOF {
		err = json.Unmarshal([]byte(buf[:n]), &res)
		if err != nil {
			fmt.Println(prefix + err.Error())
			return nil, nil
		}
	}
	s := service.Service{}
	if len(res) > 0 {
		err = json.Unmarshal([]byte(res[0].Value), &s)
		if err != nil {
			return nil, err
		}
		return &s, nil
	}
	return nil, nil
}

func UpdateService(service *service.Service) error {
	url := apiconfig.Server_URL + apiconfig.SERVICE_PATH
	fmt.Println("[tool][updateservice]: url=" + url)
	data, err := json.Marshal(service)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	fmt.Println("Response Status:", resp.Status)
	return nil
}

func UpdateDNS(dns *core.DNS) error {
	url := apiconfig.Server_URL + apiconfig.DNS_PATH
	fmt.Println("[tool][updateDNS]: url=" + url)
	data, err := json.Marshal(dns)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	fmt.Println("Response Status:", resp.Status)
	return nil
}

// TODO 讨论确定一下api-sver的rest-api用法
func DeleteService(service *service.Service) error {
	url := apiconfig.Server_URL + apiconfig.SERVICE_PATH + "?ServiceName=" + service.ServiceMeta.Name
	fmt.Println("[tool][deleteService]: url=" + url)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	fmt.Println("Response Status:", resp.Status)
	return nil
}

func GetJobFile(JobName string) core.JobUpload {
	url := apiconfig.Server_URL + apiconfig.JOB_FILE_PATH + "?JobName=" + JobName
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("[httpclient] [GetJobFile] ", "Get Job error", err)
		return core.JobUpload{}
	}
	defer resp.Body.Close()
	reader := resp.Body
	data, err := io.ReadAll(reader)
	if err != nil {
		fmt.Println("[httpclient] [GetJobFile] ", err)
		return core.JobUpload{}
	}
	var json_data string
	err = json.Unmarshal([]byte(data), &json_data)
	fmt.Println("[httpclient][GetJobFile]", json_data)
	job := core.JobUpload{}
	err = json.Unmarshal([]byte(json_data), &job)
	if err != nil {
		fmt.Println("[httpclient] [GetJobFile] ", err)
		return core.JobUpload{}
	}
	return job
}

// Get Pod

func GetPod(name string) (*core.Pod, error) {
	prefix := "[tool][GetPod]"
	fmt.Println(prefix + "key:" + name)
	url := apiconfig.Server_URL + apiconfig.POD_PATH + "?Name=" + name
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	reader := resp.Body
	data, err := io.ReadAll(reader)
	if err != nil {
		return &core.Pod{}, err
	}
	//fmt.Println("[httpclient][get pod]", string(data))
	var res []ListRes
	err = json.Unmarshal(data, &res)
	if err != nil {
		fmt.Println("[httpclient][get pod]", err)
		return &core.Pod{}, err
	}
	for _, val := range res {
		//fmt.Println("[httpclient][get pod]", res)
		pod := core.Pod{}
		err := json.Unmarshal([]byte(val.Value), &pod)
		if err != nil {
			fmt.Println("[httpclient][get pod]", err)
			return &core.Pod{}, err
		}
		return &pod, nil
	}
	return &core.Pod{}, fmt.Errorf("no such pod")
}

func UpdateDag(dag *core.DAG) error {
	url := apiconfig.Server_URL + apiconfig.WORKFLOW_PATH
	fmt.Println("[tool][updateDAG]: url=" + url)
	data, err := json.Marshal(dag)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	fmt.Println("Response Status:", resp.Status)
	return nil
}

func DeleteNode(key string) {
	strs := strings.Split(key, "/")
	nodeName := strs[4]
	url := apiconfig.Server_URL + apiconfig.NODE_PATH + "?Name=" + nodeName
	fmt.Println("[tool][deleteNode]: url=" + url)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("Response Status:", resp.Status)
	return
}

func GetTypeName(event Event) string {
	var s string
	switch event.Type {
	case Added:
		{
			s = "Add"
		}
	case Modified:
		{
			s = "Modify"
		}
	case Deleted:
		{
			s = "Delete"
		}
	case Error:
		{
			s = "Error"
		}
	}
	return s
}
