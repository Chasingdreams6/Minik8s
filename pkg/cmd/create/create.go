package create

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"minik8s/cmd/kube-apiserver/app/apiconfig"
	"minik8s/pkg/api/core"
	"minik8s/pkg/kubelet/dockerClient"
	"minik8s/pkg/service"
	"minik8s/pkg/util/random"
	"minik8s/pkg/util/web"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"gopkg.in/yaml.v3"
)

// CreateOptions is the commandline options for 'create' sub command
type CreateOptions struct {
	Filename  string
	Directory string
}

// NewCreateOptions returns an initialized CreateOptions instance
func NewCreateOptions() *CreateOptions {
	return &CreateOptions{
		Filename:  "",
		Directory: "",
	}
}

// NewCmdCreate returns new initialized instance of create sub command
func NewCmdCreate() *cobra.Command {
	o := NewCreateOptions()

	cmd := &cobra.Command{
		Use:   "create [-f FILENAME] [-d DIRECTORY]",
		Short: "Create a resource from a file or from stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.Filename == "" {
				fmt.Print("Error: must specify one -f\n")
				return errors.New("must specify one -f")
			}
			//fmt.Println(o.Filename, args)
			err := o.RunCreate(cmd, args)
			if err != nil {
				return err
			}
			return nil
		},
	}

	usage := "to use to create the resource"
	cmd.Flags().StringVarP(&o.Filename, "filename", "f", "", "filename "+usage)
	cmd.Flags().StringVarP(&o.Directory, "directory", "d", "", "directory "+usage)

	return cmd
}

// RunCreate performs the creation
func (o *CreateOptions) RunCreate(cmd *cobra.Command, args []string) error {

	if len(args) != 0 {
		return fmt.Errorf("[kubectl] [create] [RunCreate] unexpected args: %v", args)
	}

	filename := o.Filename
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("cannot open file %s", filename)
	}
	defer file.Close()
	// Read the YAML file
	var dataMap map[string]interface{}

	yamlFile, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read yaml error")
	}

	err = yaml.Unmarshal(yamlFile, &dataMap)
	if err != nil {
		fmt.Println(err)
		return err
	}
	// get kind
	var kind string
	kind = dataMap["kind"].(string)
	fmt.Println("kind:", kind)
	switch kind {
	case "Pod":
		err = o.RunCreatePod(cmd, args, yamlFile)
	case "ReplicaSet":
		err = o.RunCreateReplicaSet(cmd, args, yamlFile)
	case "Job":
		err = o.RunCreateJob(cmd, args, yamlFile)
	case "Service":
		err = o.RunCreateService(cmd, args, yamlFile)
	case "HorizontalPodAutoscaler":
		err = o.RunCreateHorizontalPodAutoscaler(cmd, args, yamlFile)
	case "DNS":
		err = o.RunCreateDNS(cmd, args, yamlFile)
	case "Workflow":
		err = o.RunCreateWorkflow(cmd, args, yamlFile)
	case "Function":
		err = o.RunCreateFunction(cmd, args, yamlFile)
	}

	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (o *CreateOptions) RunCreatePod(cmd *cobra.Command, args []string, yamlFile []byte) error {
	pod := &core.Pod{}
	err := yaml.Unmarshal(yamlFile, pod)
	if err != nil {
		return err
	}

	err = CreatePod(pod)
	if err != nil {
		return err
	}
	return nil
}

func (o *CreateOptions) RunCreateReplicaSet(cmd *cobra.Command, args []string, yamlFile []byte) error {
	r := &core.ReplicaSet{}
	err := yaml.Unmarshal(yamlFile, r)
	if err != nil {
		return err
	}

	err = CreateReplicaSet(r)
	if err != nil {
		return err
	}
	return nil
}

func (o *CreateOptions) RunCreateJob(cmd *cobra.Command, args []string, yamlFile []byte) error {
	job := &core.Job{}
	err := yaml.Unmarshal(yamlFile, job)
	if err != nil {
		return err
	}

	err = CreateJob(job)
	if err != nil {
		return err
	}
	return nil
}

func (o *CreateOptions) RunCreateDNS(cmd *cobra.Command, args []string, yamlFile []byte) error {

	r := &core.DNS{}
	err := yaml.Unmarshal(yamlFile, r)
	if err != nil {
		return err
	}
	data, err := json.Marshal(r)
	if err != nil {
		fmt.Println("[kubectl] [create] [RunCreateDNS] failed to marshal:", err)
	} else {
		//fmt.Println("[kubectl] [create] [RunCreateReplicaSet] ", string(data))
	}
	err = web.SendHttpRequest("PUT", apiconfig.Server_URL+apiconfig.DNS_PATH,
		web.WithPrefix("[kubectl] [create] [RunCreatesDNS] "),
		web.WithBody(bytes.NewBuffer(data)),
		web.WithLog(true))
	if err != nil {
		return err
	}

	return nil
}

func (o *CreateOptions) RunCreateService(cmd *cobra.Command, args []string, yamlFile []byte) error {
	s := &service.Service{}
	err := yaml.Unmarshal(yamlFile, s)
	if err != nil {
		return err
	}

	err = CreateService(s)
	if err != nil {
		return err
	}
	return nil
}

func (o *CreateOptions) RunCreateHorizontalPodAutoscaler(cmd *cobra.Command, args []string, yamlFile []byte) error {
	hpa := &core.HPA{}
	err := yaml.Unmarshal(yamlFile, hpa)
	if err != nil {
		return err
	}

	err = CreateHPA(hpa)
	if err != nil {
		return err
	}
	return nil
}

func (o *CreateOptions) RunCreateWorkflow(cmd *cobra.Command, args []string, yamlFile []byte) error {
	workflow := &core.Workflow{}
	err := yaml.Unmarshal(yamlFile, workflow)
	if err != nil {
		return err
	}
	err = CreateWorkflow(workflow)
	if err != nil {
		return err
	}
	return nil
}

func (o *CreateOptions) RunCreateFunction(cmd *cobra.Command, args []string, yamlFile []byte) error {
	function := &core.Function{}
	err := yaml.Unmarshal(yamlFile, function)
	if err != nil {
		return err
	}

	function.Spec.FileDirectory = o.Directory
	function.Spec.Image = "luhaoqi/my_module-" + function.Name + ":" + random.GenerateRandomString(4)

	err = CreateFunction(function)
	if err != nil {
		return err
	}
	return nil
}

func CreateFunction(function *core.Function) error {
	prefix := "[kubectl] [create] [CreateFunction] "
	data, err := json.Marshal(function)
	if err != nil {
		fmt.Println("[kubectl] [create] [CreateFunction] failed to marshal:", err)
	} else {
		//fmt.Println("[kubectl] [create] [RunCreateReplicaSet] ", string(data))
	}
	err = web.SendHttpRequest("PUT", apiconfig.Server_URL+apiconfig.FUNCTION_PATH,
		web.WithPrefix("[kubectl] [create] [CreateFunction] "),
		web.WithBody(bytes.NewBuffer(data)),
		web.WithLog(true))
	if err != nil {
		return err
	}

	image := function.Spec.Image

	err = dockerClient.ImageBuild(function.Spec.FileDirectory, image)
	if err != nil {
		fmt.Println(prefix, "dockerClient.ImageBuild err: ", err)
		return err
	}

	err = dockerClient.ImagePush(image)
	if err != nil {
		fmt.Println(prefix, "dockerClient.ImagePush err: ", err)
		return err
	}

	return nil
}

func CreateService(s *service.Service) error {
	data, err := json.Marshal(s)
	if err != nil {
		fmt.Println("[kubectl] [create] [CreateService] failed to marshal:", err)
	} else {
	}
	err = web.SendHttpRequest("PUT", apiconfig.Server_URL+apiconfig.SERVICE_PATH,
		web.WithPrefix("[kubectl] [create] [CreatesService] "),
		web.WithBody(bytes.NewBuffer(data)),
		web.WithLog(true))
	if err != nil {
		return err
	}
	return nil
}

func CreateReplicaSet(r *core.ReplicaSet) error {
	// 序列化 调试用法，前面多两个空格，易于阅读
	//data, err := json.MarshalIndent(pod, "", "  ")
	data, err := json.Marshal(r)
	if err != nil {
		fmt.Println("[kubectl] [create] [RunCreateReplicaSet] failed to marshal:", err)
	} else {
		//fmt.Println("[kubectl] [create] [RunCreateReplicaSet] ", string(data))
	}

	// Send a POST request to kube-apiserver to create the replicaset
	// 创建 PUT 请求
	err = web.SendHttpRequest("PUT", apiconfig.Server_URL+apiconfig.REPLICASET_PATH,
		web.WithPrefix("[kubectl] [create] [RunCreateReplicaSet] "),
		web.WithBody(bytes.NewBuffer(data)),
		web.WithLog(true))
	if err != nil {
		return err
	}

	return nil
}

func CreatePod(pod *core.Pod) error {
	// 序列化
	// 调试用法，前面多两个空格，易于阅读
	//data, err := json.MarshalIndent(pod, "", "  ")
	data, err := json.Marshal(pod)
	if err != nil {
		fmt.Println("[kubectl] [create] [RunCreatePod] failed to marshal:", err)
	} else {
		//fmt.Println("[kubectl] [create] [RunCreatePod]\n", string(data))
	}

	// Send a POST request to kube-apiserver to create the pod
	// 创建 PUT 请求
	err = web.SendHttpRequest("PUT", apiconfig.Server_URL+apiconfig.POD_PATH,
		web.WithPrefix("[kubectl] [create] [RunCreate] "),
		web.WithBody(bytes.NewBuffer(data)),
		web.WithLog(true))
	if err != nil {
		return err
	}
	return nil
}

func CreateJob(job *core.Job) error {
	// 发送job信息至apiserver保存
	job_data, err := json.Marshal(job)
	if err != nil {
		fmt.Println("marshal job error:", err)
		return fmt.Errorf("marshal job error")
	}
	job_data_resp, err := http.Post(apiconfig.Server_URL+apiconfig.JOB_PATH, "application/json", bytes.NewBuffer(job_data))
	if err != nil {
		fmt.Println("Error sending request:", err)
		return err
	}
	defer job_data_resp.Body.Close()
	if job_data_resp.StatusCode != 200 {
		return fmt.Errorf("job upload failed")
	}

	jobUpload := &core.JobUpload{}
	// 读取 job program
	program_path := job.Spec.JobTask.Program
	program, err := ioutil.ReadFile(program_path)
	if err != nil {
		return fmt.Errorf("read program error")
	}
	jobUpload.JobName = job.Spec.JobTask.JobName
	jobUpload.Program = program
	// 生成slurm文件
	jobUpload.Slurm = job.Spec.JobTask.GenerateSlurm()
	if err != nil {
		fmt.Println(err)
		return err
	}
	// 序列化
	filedata, err := json.Marshal(jobUpload)
	if err != nil {
		fmt.Println("failed to marshal gpu file:", err)
	}
	// 发送job文件至apiserver保存
	resp, err := http.Post(apiconfig.Server_URL+apiconfig.JOB_FILE_PATH, "application/json", bytes.NewBuffer(filedata))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("job file upload failed")
	}

	return nil
}

func CreateHPA(hpa *core.HPA) error {
	data, err := json.Marshal(hpa)
	if err != nil {
		fmt.Println("[kubectl] [create] [RunCreateHPA] failed to marshal:", err)
	} else {
		fmt.Println("[kubectl] [create] [RunCreateHPA] ", string(data))
	}
	err = web.SendHttpRequest("PUT", apiconfig.Server_URL+apiconfig.HPA_PATH,
		web.WithPrefix("[kubectl] [create] [RunCreateHPA] "),
		web.WithBody(bytes.NewBuffer(data)),
		web.WithLog(true))
	if err != nil {
		return err
	}
	return nil
}

func CreateWorkflow(workflow *core.Workflow) error {
	if workflow.Spec.InputData != "" {
		inputdata, err := ioutil.ReadFile(workflow.Spec.InputData)
		if err != nil {
			return fmt.Errorf("read program error")
		}
		var inputs map[string]interface{}
		err = yaml.Unmarshal(inputdata, &inputs)
		if err != nil {
			return fmt.Errorf("unmarshal input error")
		}
		data, err := json.Marshal(inputs)
		if err != nil {
			return fmt.Errorf("marshal input error")
		}
		fmt.Println("[kubectl] [create] [RunCreateWorkflow] ", string(data))
		workflow.Spec.Input = string(data)
	}
	data, err := json.Marshal(workflow)
	if err != nil {
		fmt.Println("[kubectl] [create] [RunCreateWorkflow] failed to marshal:", err)
	} else {
		fmt.Println("[kubectl] [create] [RunCreateWorkflow] ", string(data))
		// fmt.Println("[kubectl] [create] [RunCreateWorkflow] ", workflow.Spec.States[1].Choices[1].Condition.Operator)
	}
	err = web.SendHttpRequest("PUT", apiconfig.Server_URL+apiconfig.WORKFLOW_PATH,
		web.WithPrefix("[kubectl] [create] [RunCreateWorkflow] "),
		web.WithBody(bytes.NewBuffer(data)),
		web.WithLog(true))
	if err != nil {
		return err
	}
	return nil
}
