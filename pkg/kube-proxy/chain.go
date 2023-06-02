package kube_proxy

import (
	"fmt"
	"minik8s/pkg/iptables"
	"strconv"
)

func (rule *DNatRule) toSpec() {
	// table
	rule.Spec = []string{"-t", rule.Table}
	// portocol
	rule.Spec = append(rule.Spec, "-p", rule.Protocol)
	// source
	//rule.Spec = append(rule.Spec, "-s", rule.SourceIP, "--sport", rule.SourcePort)
	// dest
	rule.Spec = append(rule.Spec, "-d", rule.DestIP, "--dport", rule.DestPort)
	// action, 指做 入方向的地址转换
	rule.Spec = append(rule.Spec, "-j", "DNAT")
	// to
	rule.Spec = append(rule.Spec, "--to-destination", rule.PodIP+":"+rule.PodPort)

	fmt.Print("[chain][DNatRule][toSpec]: ")
	for _, value := range rule.Spec {
		fmt.Print(" " + value)
	}
	fmt.Println("")
}

func (chain *PodChain) toSpec() {
	chain.Spec = []string{"-p", chain.Protocol}
	chain.Spec = append(chain.Spec, "-m", "statistic")
	chain.Spec = append(chain.Spec, "--mode", "nth")
	chain.Spec = append(chain.Spec, "--every", strconv.Itoa(chain.RoundRabinNumber))
	chain.Spec = append(chain.Spec, "--packet", "0")
	chain.Spec = append(chain.Spec, "-j", chain.Name)
	fmt.Print("[chain][PodChain][toSpec]: ")
	for _, value := range chain.Spec {
		fmt.Print(" " + value)
	}
	fmt.Println("")
}

func (chain *SvcChain) toSpec() {
	chain.Spec = []string{"-s", "0/0"}
	chain.Spec = append(chain.Spec, "-p", chain.Protocol)
	chain.Spec = append(chain.Spec, "-d", chain.ClusterIp)
	chain.Spec = append(chain.Spec, "--dport", chain.ClusterPort)
	chain.Spec = append(chain.Spec, "-j", chain.Name)
	fmt.Print("[chain][svcChain][toSpec]: ")
	for _, value := range chain.Spec {
		fmt.Print(" " + value)
	}
	fmt.Println("")
}

// direct dstIp:dstPort -> PodIP:port
func NewDNatRule(PodIP string,
	port string,
	SourceIP *string,
	SourcePort *string,
	protocol string,
	father string,
	table *string,
	dstIP string,
	dstPort string) *DNatRule {
	res := &DNatRule{}
	res.PodIP = PodIP
	res.PodPort = port
	res.DestIP = dstIP
	res.DestPort = dstPort
	res.FatherChain = father
	if SourceIP != nil {
		res.SourceIP = *SourceIP
	} else {
		res.SourceIP = "0"
	}
	if SourcePort != nil {
		res.SourcePort = *SourcePort
	} else {
		res.SourcePort = "0"
	}
	res.Protocol = protocol
	if table != nil {
		res.Table = *table
	} else {
		res.Table = "nat"
	}

	res.toSpec()
	return res
}

// 根据pod-Info创建chain, father是service-chain
func NewPodChain(protocol string, pod PodInfo, table string, father string, rr int, dstIp string, dstPort string) *PodChain {
	prefix := "[chain][NewPodChain]"
	fmt.Println(prefix + " : dstIp:Port=" + dstIp + ":" + dstPort + "PodIP:Port=" + pod.IP + ":" + pod.Port)
	res := &PodChain{
		Name:             PodChainPrefix + "-" + pod.Name + pod.Port,
		Pod:              pod,
		Table:            table,
		FatherChain:      father,
		Protocol:         protocol,
		RoundRabinNumber: rr,
	}
	res.toSpec()
	ipt, err := iptables.New()
	if err != nil { // may exist before?
		fmt.Println(prefix + err.Error())
		//panic(err.Error())
	}
	err = ipt.NewChain(table, res.Name)
	if err != nil {
		fmt.Println(prefix + err.Error())
		//panic(err.Error())
	}
	res.DNatRule = NewDNatRule(pod.IP, pod.Port, nil, nil, protocol, res.Name, &table, dstIp, dstPort)
	err = res.DNatRule.ApplyRule()
	if err != nil {
		fmt.Println(prefix + err.Error())
		//panic(err.Error())
	}
	return res
}

func AddReturnToNAT(father string, table string) {
	fmt.Println("[chain][addReturn]: father=" + father)
	ipt, err := iptables.New()
	if err != nil {
		fmt.Println(err)
		return
	}
	if table == "" {
		table = "nat"
	}
	spec := []string{}
	spec = append(spec, "-p", "all")
	spec = append(spec, "-j", "RETURN")
	err = ipt.Append(table, father, spec...)
	if err != nil {
		fmt.Println(err)
	}
}

func DeleteReturnToNAT(father string, table string) {
	fmt.Println("[chain][deleteReturn]: father=" + father)
	ipt, err := iptables.New()
	if err != nil {
		fmt.Println(err)
		return
	}
	if table == "" {
		table = "nat"
	}
	spec := []string{}
	spec = append(spec, "-p", "all")
	spec = append(spec, "-j", "RETURN")
	err = ipt.Delete(table, father, spec...)
	//err = ipt.Append(table, father, spec...)
	if err != nil {
		fmt.Println(err)
	}
}

func NewSvcChain(name string, table string, father string, ip string, port string,
	protocol string, pods []*PodInfo) *SvcChain {
	prefix := "[chain][NewSvcChain]"
	fmt.Println(prefix + ": sName=" + name + " sIP:port=" + ip + ":" + port + "podNum=" + strconv.Itoa(len(pods)))
	res := &SvcChain{
		Name:        SvcChainPrefix + "-" + name + port,
		Table:       table,
		FatherChain: father,
		Protocol:    protocol,
		ClusterIp:   ip,
		ClusterPort: port,
	}
	res.toSpec()
	ipt, err := iptables.New()
	if err != nil {
		fmt.Println(prefix + err.Error())
		//panic(err.Error())
	}

	err = ipt.NewChain(table, res.Name)
	total := 1
	res.Name2Chain = make(map[string]*PodChain)
	for _, val := range pods {
		curChain := NewPodChain(protocol, *val, table, res.Name, total, ip, port)
		err = curChain.ApplyChain()
		if err != nil {
			fmt.Println(prefix + err.Error())
			//panic(err.Error())
		}
		total++
		res.Name2Chain[val.Name] = curChain
	}
	AddReturnToNAT(res.Name, "nat")
	return res
}

// ApplyRule add one rule, 增加一个规则的顶层接口
func (rule *DNatRule) ApplyRule() error {
	fmt.Println("[chain][DNAT][ApplyChain]: in")
	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	err = ipt.Append(rule.Table, rule.FatherChain, rule.Spec...)
	return err
}

// DeleteRule delete one rule
func (rule *DNatRule) DeleteRule() error {
	fmt.Println("[chain][DNAT][DeleteChain]: in")
	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	err = ipt.Delete(rule.Table, rule.FatherChain, rule.Spec...)
	return err
}

func (chain *PodChain) ApplyChain() error {
	fmt.Println("[chain][PodChain][ApplyChain]: in")
	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	//err = ipt.Append(chain.Table, chain.FatherChain, chain.Spec...)
	err = ipt.Insert(chain.Table, chain.FatherChain, 1, chain.Spec...)
	return err
}

func (chain *PodChain) DeleteChain() error {
	fmt.Println("[chain][PodChain][DeleteChain]: in")
	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	//先删除在父链中的规则
	err = ipt.Delete(chain.Table, chain.FatherChain, chain.Spec...)
	if err != nil {
		return err
	}
	//删除sep链自身中的DNat规则
	err = chain.DNatRule.DeleteRule()
	if err != nil {
		return err
	}
	//删除该链
	err = ipt.DeleteChain(chain.Table, chain.Name)
	return err
}

func (chain *SvcChain) ApplyChain() error {
	fmt.Println("[chain][SvcChain][ApplyChain]: in")
	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	err = ipt.Insert(chain.Table, chain.FatherChain, 1, chain.Spec...)
	//err = ipt.Append(chain.Table, chain.FatherChain, chain.Spec...)
	if err != nil {
		return err
	}
	return nil
}

func (chain *SvcChain) DeleteChain() error {
	fmt.Println("[chain][SvcChain][DeleteChain]: in")
	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	//删除在父链中的该规则
	err = ipt.Delete(chain.Table, chain.FatherChain, chain.Spec...)
	if err != nil {
		return err
	}
	//删除该链下的所有pod的规则
	// 需要内存中的Name2Chain数据结构支持
	for _, ch := range chain.Name2Chain {
		err = ch.DeleteChain()
		if err != nil {
			return err
		}
	}
	// 删除该链中的return链
	DeleteReturnToNAT(chain.Name, "nat")
	//删除该链
	err = ipt.DeleteChain(chain.Table, chain.Name)
	return err
}

func (chain *SvcChain) UpdateChain(newPods []*PodInfo) {
	prefix := "[chain][SvcChain][UpdateChain]"
	fmt.Println(": in")
	shouldRemain := make(map[string]*PodChain)
	for k, v := range chain.Name2Chain {
		flag := false
		for _, pod := range newPods {
			if k == pod.Name {
				flag = true
				break
			}
		}
		if flag {
			shouldRemain[k] = v // remain
		} else {
			err := v.DeleteChain()
			if err != nil {
				fmt.Println(prefix + err.Error())
				//panic(err.Error())
			}
		}
	}
	ipt, err := iptables.New()
	if err != nil {
		fmt.Println(prefix + err.Error())
		//panic(err.Error())
	}
	// first, delete all previous chain
	for _, v := range shouldRemain {
		err = ipt.Delete(v.Table, v.FatherChain, v.Spec...)
		if err != nil {
			fmt.Println(prefix + err.Error())
			//panic(err.Error())
		}
	}
	total := 1
	for _, pod := range newPods {
		podChain, ok := shouldRemain[pod.Name]
		if ok {
			podChain.RoundRabinNumber = total // update
			podChain.toSpec()
			err = ipt.Insert(podChain.Table, podChain.FatherChain, 1, podChain.Spec...)
			if err != nil {
				fmt.Println(prefix + err.Error())
				//panic(err.Error())
			}
			chain.Name2Chain[pod.Name] = podChain
		} else { // create new chain
			podChain = NewPodChain(chain.Protocol, *pod, chain.Table, chain.Name, total, chain.ClusterIp, chain.ClusterPort)
			err = podChain.ApplyChain()
			if err != nil {
				fmt.Println(prefix + err.Error())
				//panic(err.Error())
			}
			chain.Name2Chain[pod.Name] = podChain
		}
		total++
	}
}

// 启动的函数,用于创建services链并加入到OUTPUT以及PREROUTING链中
func Init() {
	ipt, err := iptables.New()
	// first, check exist
	exist, err := ipt.ChainExists("nat", "SERVICE")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if exist {
		fmt.Println("[Chain][Init] service inited before, return")
		return
	}
	err = ipt.NewChain("nat", "SERVICE")
	// Return
	AddReturnToNAT("SERVICE", "nat")

	if err != nil {
		fmt.Println(err.Error())
		//panic(err.Error())
	}
	// 增加OUTPUT链
	err = ipt.Insert("nat", "OUTPUT", 1, "-j", "SERVICE", "-s", "0/0", "-d", "0/0", "-p", "all")
	if err != nil {
		fmt.Println(err.Error())
		//panic(err.Error())
	}
	// 增加PREROUTING链
	err = ipt.Insert("nat", "PREROUTING", 1, "-j", "SERVICE", "-s", "0/0", "-d", "0/0", "-p", "all")

	if err != nil {
		fmt.Println(err.Error())
		//panic(err.Error())
	}
}
