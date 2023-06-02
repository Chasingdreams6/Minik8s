package jobcontroller

import (
	"encoding/json"
	"fmt"
	"minik8s/cmd/kube-apiserver/app/apiconfig"
	"minik8s/pkg/api/core"
	metav1 "minik8s/pkg/apis/meta/v1"
	"minik8s/pkg/client/informer"
	"minik8s/pkg/client/tool"
	q "minik8s/pkg/util/concurrentqueue"
	"time"
)

type JobController struct {
	JobInformer informer.Informer
	queue       *q.ConcurrentQueue
}

func NewJobController() *JobController {
	return &JobController{
		JobInformer: informer.NewInformer(apiconfig.JOB_PATH),
		queue:       q.NewConcurrentQueue(),
	}
}

func (jc *JobController) Register() {
	jc.JobInformer.AddEventHandler(tool.Added, jc.AddJob)
	jc.JobInformer.AddEventHandler(tool.Deleted, jc.DeleteJob)
}

func (jc *JobController) AddJob(event tool.Event) {
	fmt.Println("[jobcontroller][AddJob] add job")
	var job *core.Job
	jc.JobInformer.Set(event.Key, event.Val)
	err := json.Unmarshal([]byte(event.Val), &job)
	if err != nil {
		fmt.Println("[jobcontroller] add job error")
		return
	}
	jc.queue.Push(job)
}

func (jc *JobController) DeleteJob(event tool.Event) {
	fmt.Println("[jobcontroller][DeleteJob] delete job")
	var job *core.Job
	val := jc.JobInformer.Get(event.Key)
	jc.JobInformer.Delete(event.Key)
	err := json.Unmarshal([]byte(val), &job)
	if err != nil {
		fmt.Println("[jobcontroller] delete job error")
		return
	}
	tool.DeletePod(job.Name)
}

func (jc *JobController) worker() {
	for {
		if !jc.queue.IsEmpty() {
			job := jc.queue.Pop()
			jc.RunJob(job.(*core.Job))
		} else {
			time.Sleep(time.Second)
		}

	}
}

func (jc *JobController) RunJob(job *core.Job) {
	cmd_job := fmt.Sprintf("--jobname=%s", job.Name)
	cmd_out := fmt.Sprintf("--outfile=%s", job.Name)
	cmd_err := fmt.Sprintf("--errfile=%s", job.Name)
	jobcontainer := core.Container{
		Name:  "gpu",
		Image: "gpu-jobs-image",
		VolumeMounts: []core.VolumeMount{
			{
				Name:      "job-volume",
				MountPath: "/home/job",
			},
		},
		EntryPoint: []string{"./gpuserver"},
		Command:    []string{cmd_job, cmd_out, cmd_err},
	}

	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: job.Name,
		},
		Spec: core.PodSpec{
			Volumes: []core.Volume{
				{
					Name:     "job-volume",
					HostPath: "/home/job",
				},
			},
			Containers: []core.Container{
				jobcontainer,
			},
			GPUJob:     true,
			GPUJobName: job.Name,
		},
	}
	tool.AddPod(pod)
}

func (jc *JobController) Run() {
	go jc.JobInformer.Run()
	go jc.worker()
	select {}
}
