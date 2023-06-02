package main

import (
	"flag"
	kubeproxy "minik8s/pkg/kube-proxy"
)

func main() {
	MasterIP := flag.String("masterip", "", "master ip")
	flag.Parse()
	proxy := kubeproxy.NewKubeProxy(*MasterIP)
	proxy.Register()
	proxy.Run()
	select {}
}
