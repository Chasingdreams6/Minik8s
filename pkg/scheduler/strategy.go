package scheduler

import (
	"fmt"
	"math/rand"
	"minik8s/pkg/api/core"
	"time"
)

var global_cnt int = 0

func roundrobin_strategy(nodes []core.Node) string {
	global_cnt = (global_cnt + 1) % len(nodes)
	fmt.Println("[scheduler] [roundrobin_strategy] nodes:", nodes, "global_cnt:", global_cnt)
	return nodes[global_cnt].Name
}

func random_strategy(nodes []core.Node) string {
	rand.Seed(time.Now().UnixNano())
	num := rand.Intn(len(nodes))
	return nodes[num].Name
}
