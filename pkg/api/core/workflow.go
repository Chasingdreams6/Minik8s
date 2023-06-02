package core

import (
	"fmt"
	metav1 "minik8s/pkg/apis/meta/v1"
	"minik8s/pkg/types"
	"minik8s/pkg/util/random"
)

// workflow
type Workflow struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec              WorkflowSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

type WorkflowSpec struct {
	InputData string  `json:"InputData,omitempty" yaml:"InputData,omitempty"`
	States    []State `json:"states,omitempty" yaml:"states,omitempty"`
	Input     string  `json:"input,omitempty" yaml:"input,omitempty"`
	Result    string  `json:"result,omitempty" yaml:"result,omitempty"`
}

type State struct {
	Name     string    `json:"Name" yaml:"Name"`
	Type     StateType `json:"Type" yaml:"Type"`
	Choices  []Choice  `json:"Choices,omitempty" yaml:"Choices,omitempty"`
	Resource string    `json:"Resource,omitempty" yaml:"Resource,omitempty"`
	Next     string    `json:"Next,omitempty" yaml:"Next,omitempty"`
	End      bool      `json:"End,omitempty" yaml:"End,omitempty"`
}

type StateType string

const (
	StateTypeTask   StateType = "Task"
	StateTypeChoice StateType = "Choice"
	StateTypeInput  StateType = "Input"
	StateTypeEnd    StateType = "End"
)

type Choice struct {
	Condition ChoiceCondition `json:"Condition" yaml:"Condition"`
	Next      string          `json:"Next" yaml:"Next"`
}

type ChoiceCondition struct {
	Type     ConditionType `json:"Type" yaml:"Type"`
	Variable string        `json:"Variable" yaml:"Variable"`
	Operator string        `json:"Operator" yaml:"Operator"`
	Value    int           `json:"Value" yaml:"Value"`
}

type ConditionType string

const (
	ConditionTypeNumeric = "Numeric"
	ConditionTypeTrue    = "True"
)

// DAG

type DAG struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	StartNode         DAGNode
	Input             string
	Nodes             []DAGNode            `json:"nodes,omitempty" yaml:"nodes,omitempty"`
	Edges             map[string][]DAGEdge `json:"edges,omitempty" yaml:"edges,omitempty"`
	Result            string               `json:"result,omitempty" yaml:"result,omitempty"`
}

type DAGNode struct {
	Type     StateType `json:"type" yaml:"type"`
	UID      types.UID `json:"uid" yaml:"uid"`
	Function Function  `json:"function" yaml:"function"`
}

type DAGEdge struct {
	From      DAGNode         `json:"from" yaml:"from"`
	To        DAGNode         `json:"to" yaml:"to"`
	Condition ChoiceCondition `json:"condition" yaml:"condition"`
}

func (w *Workflow) Workflow2DAG(getfunc func(string) (Function, error)) (*DAG, error) {
	var dag DAG
	dag.Edges = make(map[string][]DAGEdge)
	mapState := make(map[string]State)
	mapNode := make(map[string]*DAGNode)
	for _, state := range w.Spec.States {
		var node *DAGNode
		_, ok := mapState[state.Name]
		if ok {
			return nil, fmt.Errorf("state name %s is duplicate", state.Name)
		} else {
			mapState[state.Name] = state
		}
		if state.Type == StateTypeInput {
			dag.StartNode = DAGNode{
				Type: StateTypeInput,
				UID:  random.GenerateUUID(),
			}
			dag.Nodes = append(dag.Nodes, dag.StartNode)
			node = &dag.StartNode
		}
		if state.Type == StateTypeTask {
			nodeType := StateTypeTask
			if state.End {
				nodeType = StateTypeEnd
			}
			function, err := getfunc(state.Resource)
			if err != nil {
				return nil, err
			}
			node = &DAGNode{
				Type:     nodeType,
				Function: function,
				UID:      random.GenerateUUID(),
			}
			dag.Nodes = append(dag.Nodes, *node)
		}
		if state.Type == StateTypeChoice {
			node = &DAGNode{
				Type: StateTypeChoice,
				UID:  random.GenerateUUID(),
			}
			dag.Nodes = append(dag.Nodes, *node)
		}
		mapNode[state.Name] = node
	}
	for _, state := range w.Spec.States {
		curNode := mapNode[state.Name]
		if state.Type != StateTypeChoice {
			if state.Next != "" {
				if state.End {
					return nil, fmt.Errorf("state name %s is end state, next is not allowed", state.Name)
				}
				nextNode, ok := mapNode[state.Next]
				if ok {
					edge := DAGEdge{
						From:      *curNode,
						To:        *nextNode,
						Condition: ChoiceCondition{Type: ConditionTypeTrue},
					}
					dag.Edges[string(curNode.UID)] = append(dag.Edges[string(curNode.UID)], edge)
				} else {
					return nil, fmt.Errorf("state name %s is not exist", state.Next)
				}
			}
		} else {
			for _, choice := range state.Choices {
				nextNode, ok := mapNode[choice.Next]
				if ok {
					edge := DAGEdge{
						From:      *curNode,
						To:        *nextNode,
						Condition: choice.Condition,
					}
					dag.Edges[string(curNode.UID)] = append(dag.Edges[string(curNode.UID)], edge)
				} else {
					return nil, fmt.Errorf("state name %s is not exist", choice.Next)
				}
			}
		}
	}
	return &dag, nil
}
