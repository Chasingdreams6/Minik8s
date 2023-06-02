package workflowcontroller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"minik8s/cmd/kube-apiserver/app/apiconfig"
	"minik8s/pkg/api/core"
	"minik8s/pkg/client/informer"
	"minik8s/pkg/client/tool"
	q "minik8s/pkg/util/concurrentqueue"
	"minik8s/pkg/util/web"
	"time"
)

type WorkflowController struct {
	WorkflowInformer informer.Informer
	queue            *q.ConcurrentQueue
	mark             map[string]bool
}

func NewWorkflowController() *WorkflowController {
	return &WorkflowController{
		WorkflowInformer: informer.NewInformer(apiconfig.WORKFLOW_PATH),
		queue:            q.NewConcurrentQueue(),
		mark:             make(map[string]bool),
	}
}

func (wfc *WorkflowController) Register() {
	wfc.WorkflowInformer.AddEventHandler(tool.Added, wfc.AddWorkflow)
	wfc.WorkflowInformer.AddEventHandler(tool.Modified, wfc.AddWorkflow)
	wfc.WorkflowInformer.AddEventHandler(tool.Deleted, wfc.DeleteWorkflow)
}

func (wfc *WorkflowController) Run() {
	go wfc.WorkflowInformer.Run()
	go worker(wfc)
	select {}
}

func worker(wfc *WorkflowController) {
	prefix := "[workflow] "
	for {
		if !wfc.queue.IsEmpty() {
			key := wfc.queue.Pop().(string)
			val, exist := (*wfc.WorkflowInformer.GetCache())[key]
			if !exist {
				fmt.Println("[ERROR] ", prefix, "cache doesn't have key:", key)
				continue
			}
			dag := &core.DAG{}
			err := json.Unmarshal([]byte(val), dag)
			if err != nil {
				fmt.Println("[ERROR] ", prefix, err)
				return
			}
			if mark, exist := wfc.mark[string(dag.UID)]; exist && mark {
				fmt.Println("[INFO] ", prefix, "dag have been runned")
				continue
			}
			go wfc.DoWorkflowDAG(dag)
		} else {
			time.Sleep(1 * time.Second)
		}
	}
}

func (wfc *WorkflowController) DoWorkflowDAG(dag *core.DAG) {
	fmt.Println("DoWorkflow")
	wfc.mark[string(dag.UID)] = true
	startnode := dag.StartNode
	curnode := startnode
	var result string
	var err error
	for {
		if curnode.Type == core.StateTypeInput {
			result = dag.Input
		} else if curnode.Type == core.StateTypeTask || curnode.Type == core.StateTypeEnd {
			result, err = TriggerFunc(curnode.Function, result)
			if err != nil {
				fmt.Println("[workflow] running func:", err)
				dag.Result = "running func: " + curnode.Function.Name + " error: " + err.Error()
				tool.UpdateDag(dag)
				return
			}
			if result == "" {
				time.Sleep(1 * time.Second)
				continue
			}
		}
		// if end node, break
		if curnode.Type == core.StateTypeEnd {
			fmt.Println("[workflow] end node")
			dag.Result = result
			tool.UpdateDag(dag)
			return
		}
		// whether should break up to the functional requirement
		success := false
		for _, edge := range dag.Edges[string(curnode.UID)] {
			if edge.Condition.Type == core.ConditionTypeTrue {
				curnode = edge.To
				success = true
				break
			} else if evalCondition(edge.Condition, result) {
				curnode = edge.To
				success = true
				break
			}
		}
		if !success {
			fmt.Println("[workflow] no edge can be triggered")
			dag.Result = "no edge can be triggered"
			tool.UpdateDag(dag)
			return
		}
	}
}

// func condition
func evalCondition(condition core.ChoiceCondition, result string) bool {
	var cond map[string]interface{}
	err := json.Unmarshal([]byte(result), &cond)
	if err != nil {
		fmt.Println("[workflow] evalCondition:", err)
		return false
	}
	val, ok := cond[condition.Variable]
	if !ok {
		fmt.Println("[workflow] evalCondition: no variable:", condition.Variable)
		return false
	}
	fmt.Println("[workflow] evalCondition: val:", val, "Operator:", condition.Operator)
	switch condition.Operator {
	case "==":
		return int(val.(float64)) == condition.Value
	case "!=":
		return int(val.(float64)) != condition.Value
	case ">":
		return int(val.(float64)) > condition.Value
	case ">=":
		return int(val.(float64)) >= condition.Value
	case "<":
		return int(val.(float64)) < condition.Value
	case "<=":
		return int(val.(float64)) <= condition.Value
	}
	fmt.Println("[workflow] evalCondition: no operator", condition.Operator)
	return false
}

// func invoke
func TriggerFunc(f core.Function, input string) (string, error) {
	url := apiconfig.Server_URL + "/invoke/" + f.Name
	fmt.Println("url: ", url)
	bodyBytes := make([]byte, 4096)
	fmt.Println("[workflow][TriggerFunc]input: ", input)
	web.SendHttpRequest("POST", url, web.WithBody(bytes.NewBuffer([]byte(input))), web.WithBodyBytes(&bodyBytes))
	fmt.Println("[workflow][TriggerFunc]TriggerFunc: ", f.Name)
	fmt.Println("[workflow][TriggerFunc]TriggerFuncResult: ", string(bodyBytes))
	return string(bodyBytes), nil
}

func (wfc *WorkflowController) AddWorkflow(event tool.Event) {
	prefix := "[Workflow] [AddWorkflow] "
	fmt.Println(prefix, "event.type: ", tool.GetTypeName(event))
	fmt.Println(prefix, "event.Key: ", event.Key)
	wfc.WorkflowInformer.Set(event.Key, event.Val)
	wfc.queue.Push(event.Key)
}

func (wfc *WorkflowController) DeleteWorkflow(event tool.Event) {
	prefix := "[Workflow] [DeleteWorkflow] "
	fmt.Println(prefix, "event.type: ", tool.GetTypeName(event))
	fmt.Println(prefix, "event.Key: ", event.Key)
	wfc.WorkflowInformer.Delete(event.Key)
}
