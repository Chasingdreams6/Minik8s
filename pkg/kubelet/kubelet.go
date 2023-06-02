package kubelet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"minik8s/cmd/kube-apiserver/app/apiconfig"
	kube_proxy "minik8s/configs"
	"minik8s/pkg/api/core"
	"minik8s/pkg/client/informer"
	"minik8s/pkg/client/tool"
	"minik8s/pkg/kubelet/dockerClient"
	"minik8s/pkg/kubelet/monitor"
	"minik8s/pkg/util/file"
	"minik8s/pkg/util/web"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Kubelet struct {
	Monitor      *monitor.Monitor
	PodInformer  informer.Informer
	NodeInformer informer.Informer
	//FunctionInformer informer.Informer
	node     core.Node
	workers  map[string]time.Time
	masterIp string
}

func NewKubelet(name string, nodeIp string, masterIp string) (*Kubelet, error) {
	node := core.Node{}
	node.Name = name
	node.Spec.NodeIP = nodeIp
	apiconfig.Server_URL = masterIp
	node.Labels = make(map[string]string)
	parseUrl, err := url.Parse(masterIp)
	tmp := parseUrl.Hostname()
	if tmp == nodeIp { // is master
		node.Labels["kind"] = "Master"
		fmt.Println("I am Master")
	} else {
		node.Labels["kind"] = "Worker"
		fmt.Println("I am Worker")
	}
	err = tool.AddNode(&node)
	if err != nil {
		return nil, err
	}
	return &Kubelet{
		Monitor:     monitor.NewMonitor(9400, &node),
		PodInformer: informer.NewInformer(apiconfig.POD_PATH),
		//FunctionInformer: informer.NewInformer(apiconfig.FUNCTION_PATH),
		node:         node,
		NodeInformer: informer.NewInformer(apiconfig.NODE_PATH),
		workers:      make(map[string]time.Time),
		masterIp:     tmp,
	}, nil
}

func (k *Kubelet) AddNodeHandler(event tool.Event) {
	prefix := "[Kubelet][AddNodeHandler]"
	if k.node.Labels["kind"] == "Master" {
		if event.Key == apiconfig.NODE_PATH+"/"+"node1" { // TODO, master must be node1
			return
		}
		fmt.Println(prefix + "Add worker to master's worker map")
		k.workers[event.Key] = time.Now()
	}
}

// this should be executed by a thread
func (k *Kubelet) MasterChecker() {
	for {
		var des []string
		for k, v := range k.workers {
			fmt.Println("[Kubelet][MasterChecker], key:=" + k)
			cur := time.Now()
			dur := cur.Sub(v)
			if dur > 10*time.Second { // deleteNode in database
				fmt.Println("[Kubelet][MasterChecker], delete key:=" + k)
				tool.DeleteNode(k)
				des = append(des, k)
			}
		}
		for _, v := range des {
			delete(k.workers, v)
		}
		time.Sleep(10 * time.Second)
	}
}

func (k *Kubelet) DeleteNodeHandler(event tool.Event) {
	prefix := "[Kubelet][DeleteNodeHandler]"
	if k.node.Labels["kind"] == "Master" {
		if event.Key == apiconfig.NODE_PATH+"/"+"node1" { // TODO, master must be node1
			return
		}
		fmt.Println(prefix + "Delete worker in master")
		delete(k.workers, event.Key)
	}
}

func (k *Kubelet) Register() {
	k.PodInformer.AddEventHandler(tool.Added, k.CreatePod)
	k.PodInformer.AddEventHandler(tool.Modified, k.UpdatePod)
	k.PodInformer.AddEventHandler(tool.Deleted, k.DeletePod)
	//k.FunctionInformer.AddEventHandler(tool.Added, k.CreateFunction)
	//k.FunctionInformer.AddEventHandler(tool.Modified, k.UpdateFunction)
	//k.FunctionInformer.AddEventHandler(tool.Deleted, k.DeleteFunction)
}

//func (k *Kubelet) CreateFunction(event tool.Event) {
//	prefix := "[kubelet] [CreateFunction] "
//	// handle event
//	fmt.Println(prefix, "event.Type: ", tool.GetTypeName(event))
//	fmt.Println(prefix, "event.Key: ", event.Key)
//	k.FunctionInformer.Set(event.Key, event.Val)
//
//}
//
//func (k *Kubelet) UpdateFunction(event tool.Event) {
//	prefix := "[kubelet] [UpdateFunction] "
//	// handle event
//	fmt.Println(prefix, "event.Type: ", tool.GetTypeName(event))
//	fmt.Println(prefix, "event.Key: ", event.Key)
//	k.FunctionInformer.Set(event.Key, event.Val)
//
//	function := &core.Function{}
//	err := json.Unmarshal([]byte(event.Val), function)
//	if err != nil {
//		fmt.Println(prefix, "json.Unmarshal err: ", err)
//		return
//	}
//
//	// 镜像会自动覆盖
//
//	//err = dockerClient.ImageBuild(function.Spec.FileDirectory, "luhaoqi/my_module:"+function.Name)
//	//if err != nil {
//	//	fmt.Println(prefix, "dockerClient.ImageBuild err: ", err)
//	//	return
//	//}
//
//	// TODO: 删除所有之前Function的容器
//}
//
//func (k *Kubelet) DeleteFunction(event tool.Event) {
//	prefix := "[kubelet] [DeleteFunction] "
//	// handle event
//	fmt.Println(prefix, "event.Type: ", tool.GetTypeName(event))
//	fmt.Println(prefix, "event.Key: ", event.Key)
//	val := k.FunctionInformer.Get(event.Key)
//	k.FunctionInformer.Delete(event.Key)
//
//	function := &core.Function{}
//	err := json.Unmarshal([]byte(val), function)
//	if err != nil {
//		fmt.Println(prefix, "json.Unmarshal err: ", err)
//		return
//	}
//
//	// TODO: image remove 目前还是空的
//	err = dockerClient.ImageRemove("my_module:" + function.Name)
//	if err != nil {
//		fmt.Println(prefix, "dockerClient.ImageRemove err: ", err)
//		return
//	}
//}

// this should be a thread
func (k *Kubelet) HandleConnection(conn net.Conn) {
	prefix := "[Kubelet][HandleConnection]"
	buf := make([]byte, 1024)
	fmt.Println(prefix + "server connected a client " + conn.RemoteAddr().Network())
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println(prefix + err.Error())
			break
		}
		fmt.Println(prefix + "got heartbeart from key:" + string(buf[:n]))
		key := string(buf[:n])
		k.workers[key] = time.Now() // update
	}
	fmt.Println(prefix + "Connection closed!")
}

func (k *Kubelet) HeartBeatServer() {
	prefix := "[Kubelet][HeartBeatServer]"
	ln, err := net.Listen("tcp", ":12345")
	if err != nil {
		fmt.Println(prefix + err.Error())
		return
	}
	defer ln.Close()
	fmt.Println(prefix + "Started")
	for {
		conn, err := ln.Accept() // waiting for a connection
		if err != nil {
			fmt.Println(prefix + err.Error())
			continue
		}
		go k.HandleConnection(conn)
	}
}

func (k *Kubelet) HeartBeatClient() { // linking and send
	prefix := "[Kubelet][HeartBeatClient]"
	defer func() { // handler error, not exit
		if err := recover(); err != nil {
			fmt.Println(err)
			time.Sleep(2 * time.Second)
			go k.HeartBeatClient()
		}
	}()
	for {
		conn, err := net.Dial("tcp", k.masterIp+":12345")
		if err != nil {
			fmt.Println(prefix + err.Error())
		}
		fmt.Println("[Kubelet][HeartBeatClient]:" + "sent hb key:" + k.node.Name)
		_, err1 := conn.Write([]byte(apiconfig.NODE_PATH + "/" + k.node.Name))
		if err1 != nil {
			fmt.Println(prefix + err1.Error())
		}
		time.Sleep(5 * time.Second)
		conn.Close()
	}
}

func (k *Kubelet) CreatePod(event tool.Event) {
	prefix := "[kubelet] [CreatePod] "
	// handle event
	fmt.Println(prefix, "event.Type: ", tool.GetTypeName(event))
	fmt.Println(prefix, "event.Key: ", event.Key)
	fmt.Println(prefix, "event.Val: ", event.Val)
	k.PodInformer.Set(event.Key, event.Val)

}

func (k *Kubelet) UpdatePod(event tool.Event) {
	prefix := "[kubelet] [UpdatePod] "
	// handle event
	fmt.Println(prefix, "event.Type: ", tool.GetTypeName(event))
	fmt.Println(prefix, "event.Key: ", event.Key)
	fmt.Println(prefix, "event.Val: ", event.Val)
	k.PodInformer.Set(event.Key, event.Val)

	pod := &core.Pod{}
	err := json.Unmarshal([]byte(event.Val), pod)
	if err != nil {
		fmt.Println(prefix, err)
		return
	}

	if pod.Spec.NodeName != k.node.Name {
		fmt.Println(prefix, "Not mine. The node of the pod is :", pod.Spec.NodeName)
		return
	}

	if pod.Status.Phase != "Pending" {
		fmt.Println(prefix, "phase is not satisfied:", pod.Status.Phase)
		return
	}

	// Gateway's pod is specific, it's name's can't be changed
	if !strings.Contains(pod.ObjectMeta.Name, kube_proxy.GatewayPodPrefix) {
		for i, v := range pod.Spec.Containers {
			pod.Spec.Containers[i].Name = pod.Name + "-" + v.Name
		}
	}

	// 判断创建pod是否是gpu类型
	if pod.Spec.GPUJob == true {
		// 获取gpu运行文件至job目录
		fmt.Println(prefix, "get gpufiles")
		err = GetGpuJobFile(pod.Spec.GPUJobName)
		if err != nil {
			fmt.Println(prefix, "[ERROR]", err)
			return
		}
	}

	var metaData []core.ContainerMeta
	var netSetting *types.NetworkSettings
	var err1 error
	if pod.ObjectMeta.Name == kube_proxy.CoreDNSPodName {
		metaData, netSetting, err1 = dockerClient.CreateCoreDNSPod(*pod)
	} else {
		metaData, netSetting, err1 = dockerClient.CreatePod(*pod)
	}
	if err1 != nil {
		fmt.Println(err1)
	}
	// 创建成功 修改Status
	pod.Status.Phase = core.PodRunning

	// 更新podIp
	pod.Status.PodIP = netSetting.IPAddress

	data, err := json.Marshal(pod)
	if err != nil {
		fmt.Println(prefix, "failed to marshal:", err)
	}

	err = web.SendHttpRequest("POST", apiconfig.Server_URL+apiconfig.POD_PATH,
		web.WithPrefix(prefix), web.WithBody(bytes.NewBuffer(data)))
	if err != nil {
		return
	}

	for _, meta := range metaData {
		fmt.Println(meta.Name, meta.Id)
	}
	net, err := json.MarshalIndent(netSetting, "", "  ")
	fmt.Println("net: ")
	fmt.Println(string(net))
	fmt.Println("-----------")
}

func (k *Kubelet) DeletePod(event tool.Event) {
	// handle event
	fmt.Println("In DeletePod EventHandler:")
	fmt.Println("event.Key: ", event.Key)
	key := k.PodInformer.Get(event.Key)
	fmt.Println("event.Val: ", key)
	k.PodInformer.Delete(event.Key)

	pod := &core.Pod{}
	err := json.Unmarshal([]byte(key), pod)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = dockerClient.DeletePod(*pod)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func GetGpuJobFile(jobname string) error {
	// 获取gpu运行文件至job目录
	jobFile := tool.GetJobFile(jobname)
	if len(jobFile.Program) == 0 || len(jobFile.Slurm) == 0 {
		return fmt.Errorf("[kubelet] [UpdatePod] get gpufiles empty")
	}
	// get program
	program_name := jobFile.JobName + ".cu"
	err := file.MakeFile(jobFile.Program, program_name, apiconfig.JOB_FILE_DIR_PATH+"/"+jobFile.JobName)
	if err != nil {
		return err
	}
	// get slurm
	slurm_name := jobFile.JobName + ".slurm"
	err = file.MakeFile(jobFile.Slurm, slurm_name, apiconfig.JOB_FILE_DIR_PATH+"/"+jobFile.JobName)
	if err != nil {
		return err
	}
	return nil
}

func (k *Kubelet) Listener(needLog bool) {
	re := regexp.MustCompile(`^Exited \((\d+)\)`)

	// docker ps every 15 seconds
	for {
		containers, err := dockerClient.GetAllContainers()
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, con := range containers {
			match := re.FindStringSubmatch(con.Status)
			Exited := false
			ExitedValue := ""
			if len(match) > 1 {
				if needLog {
					fmt.Printf("%v Exited:%s\n", con.Names[0][1:], match[1])
				}
				Exited = true
				ExitedValue = match[1]
			} else {
				if needLog {
					fmt.Printf("%v not found.\n", con.Names[0][1:])
				}
			}
			if Exited {
				pod_cache := k.PodInformer.GetCache()
				for _, v := range *pod_cache {
					pod := &core.Pod{}
					err := json.Unmarshal([]byte(v), pod)
					if err != nil {
						fmt.Println(err)
						return
					}
					for _, c := range pod.Spec.Containers {
						if c.Name == con.Names[0][1:] && pod.Status.Phase == "Running" {
							fmt.Println("pod:", pod.Name, " container:", c.Name, " Exited With", ExitedValue)
							if ExitedValue == "0" {
								pod.Status.Phase = "Succeeded"
							} else {
								pod.Status.Phase = "Failed"
							}

							data, err := json.Marshal(pod)
							if err != nil {
								fmt.Println(err)
								return
							}
							err = web.SendHttpRequest("POST", apiconfig.Server_URL+apiconfig.POD_PATH,
								web.WithPrefix("[kubelet] [Listener]"), web.WithBody(bytes.NewBuffer(data)))
							if err != nil {
								fmt.Println(err)
								return
							}
							if needLog {
								fmt.Printf("%v update status to %s.\n", con.Names[0][1:], pod.Status.Phase)
							}
							break
						}
					}

				}
			}
		}
		if needLog {
			fmt.Println("-----------")
		}
		time.Sleep(15 * time.Second)
	}
}

func (k *Kubelet) Run() {
	go k.Monitor.Run()
	go k.PodInformer.Run()
	//go k.FunctionInformer.Run()
	go k.Listener(true)
	if k.node.Labels["kind"] == "Master" {
		go k.HeartBeatServer()
		go k.MasterChecker()
	} else {
		go k.HeartBeatClient()
	}
	select {}
}
