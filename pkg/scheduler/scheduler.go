package scheduler

import (
	"encoding/json"
	"fmt"
	"minik8s/cmd/kube-apiserver/app/apiconfig"
	"minik8s/pkg/api/core"
	"minik8s/pkg/client/informer"
	"minik8s/pkg/client/tool"
	q "minik8s/pkg/util/concurrentqueue"
	"sort"
	"time"
)

type Scheduler struct {
	PodInformer  informer.Informer
	NodeInformer informer.Informer
	Strategy     string
	queue        *q.ConcurrentQueue
}

const (
	RandomStrategy string = "RandomStrategy"
	RRStrategy     string = "RRStrategy"
)

func NewScheduler(Strategy *string) *Scheduler {
	return &Scheduler{
		PodInformer:  informer.NewInformer(apiconfig.POD_PATH),
		NodeInformer: informer.NewInformer(apiconfig.NODE_PATH),
		queue:        q.NewConcurrentQueue(),
		Strategy:     *Strategy,
	}
}

func (s *Scheduler) Register() {
	s.PodInformer.AddEventHandler(tool.Added, s.AddPod)
	s.NodeInformer.AddEventHandler(tool.Added, s.AddNode)
	s.NodeInformer.AddEventHandler(tool.Deleted, s.DeleteNode)
}

func (s *Scheduler) AddNode(event tool.Event) {
	fmt.Println("[scheduler] [AddNode]")
	s.NodeInformer.Set(event.Key, event.Val)
}

func (s *Scheduler) DeleteNode(event tool.Event) {
	fmt.Println("[scheduler] [DeleteNode]")
	s.NodeInformer.Delete(event.Key)
}

func (s *Scheduler) AddPod(event tool.Event) {
	fmt.Println("[scheduler] [AddPod] add pod")
	s.PodInformer.Set(event.Key, event.Val)
	pod := &core.Pod{}
	err := json.Unmarshal([]byte(event.Val), &pod)
	if err != nil {
		return
	}
	if pod.Spec.NodeName != "" {
		fmt.Println("[scheduler][schedule] pod" + pod.Name + " should on " + pod.Spec.NodeName)
		err := tool.UpdatePod(pod)
		if err != nil {
			fmt.Println("[scheduler][schedule] addPod" + err.Error())
			return
		}
		return
	}
	s.queue.Push(pod)
}

func (s *Scheduler) worker() {
	for {
		//TODO: optmize can use channel or condition variable
		if !s.queue.IsEmpty() {
			pod := s.queue.Pop()
			s.Schedule(pod.(*core.Pod))
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (s *Scheduler) Schedule(pod *core.Pod) {
	node := s.GetNode()
	var nodeName string
	if s.Strategy == RRStrategy {
		nodeName = roundrobin_strategy(node)
	} else {
		nodeName = random_strategy(node)
	}
	pod.Spec.NodeName = nodeName
	fmt.Println("[scheduler] [Schedule] schedule "+pod.Name+"to node:", nodeName)
	err := tool.UpdatePod(pod)
	if err != nil {
		return
	}
}

func (s *Scheduler) GetNode() []core.Node {
	// TODO : can optimize by cache
	res := s.NodeInformer.GetCache()
	var nodes []core.Node
	for _, val := range *res {
		node := core.Node{}
		err := json.Unmarshal([]byte(val), &node)
		if err != nil {
			continue
		}
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})
	return nodes
}

func (s *Scheduler) Run() {
	go s.PodInformer.Run()
	go s.NodeInformer.Run()
	go s.worker()
	select {}
}
