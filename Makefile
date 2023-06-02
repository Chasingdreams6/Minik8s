clean:
	iptables-restore < /root/iptable_ori
	etcdctl del "/api" --prefix
	#rm -rf ./bin 2>/dev/null || true
	pwd
	rm -rf /root/nginx 2>/dev/null || true
	docker ps -aq --filter "name=^coreDNS" | xargs -r docker stop
	docker ps -aq --filter "name=^coreDNS" | xargs -r docker rm
	docker volume rm volume-coredns 2>/dev/null || true

# deprecated
construct:
	mkdir -p bin
	/usr/local/go/bin/go build -o ./bin/kube-apiserver ./cmd/kube-apiserver/kube-apiserver.go
	/usr/local/go/bin/go build -o ./bin/kube-scheduler ./cmd/kube-scheduler/kube-scheduler.go
	/usr/local/go/bin/go build -o ./bin/kubelet ./cmd/kubelet/kubelet.go
	/usr/local/go/bin/go build -o ./bin/kube-controller-manager ./cmd/kube-controller-manager/kube-controller-manager.go
	/usr/local/go/bin/go build -o ./bin/kubeproxy ./cmd/kube-proxy/kubeproxy.go
	/usr/local/go/bin/go build -o ./bin/kubeservice ./cmd/kube-service/kubeservice.go
	/usr/local/go/bin/go build -o ./bin/kubectl ./cmd/kubectl/kubectl.go

# deprecated
build:
	/usr/local/go/bin/go build -o ./binx/kube-apiserver ./cmd/kube-apiserver/kube-apiserver.go
	/usr/local/go/bin/go build -o ./binx/kube-scheduler ./cmd/kube-scheduler/kube-scheduler.go
	/usr/local/go/bin/go build -o ./binx/kubelet ./cmd/kubelet/kubelet.go
	/usr/local/go/bin/go build -o ./binx/kube-controller-manager ./cmd/kube-controller-manager/kube-controller-manager.go
	/usr/local/go/bin/go build -o ./binx/kubeproxy ./cmd/kube-proxy/kubeproxy.go
	/usr/local/go/bin/go build -o ./binx/kubeservice ./cmd/kube-service/kubeservice.go

run:
	mkdir -p /root/nginx
	/usr/local/go/bin/go run ./cmd/kube-apiserver/kube-apiserver.go > log/apiserver.log &
	/usr/local/go/bin/go run ./cmd/kube-scheduler/kube-scheduler.go --strategy=RRStrategy > log/scheduler.log &
	/usr/local/go/bin/go run ./cmd/kubelet/kubelet.go --nodename=node1 --nodeip=192.168.1.7 --masterip=http://192.168.1.7:8080 > log/kubelet.log &
	/usr/local/go/bin/go run ./cmd/kube-controller-manager/kube-controller-manager.go > log/controller-manager.log &
	/usr/local/go/bin/go run ./cmd/kube-proxy/kubeproxy.go --masterip=http://192.168.1.7:8080 > log/kubeproxy.log &
	/usr/local/go/bin/go run ./cmd/kube-service/kubeservice.go > log/kubeservice.log &

m3:
	go run ./cmd/kubelet/kubelet.go --nodename=node3 --nodeip=192.168.1.11 --masterip=http://192.168.1.7:8080 > log/kubelet-m3.log &
	/usr/local/go/bin/go run ./cmd/kube-proxy/kubeproxy.go --masterip=http://192.168.1.7:8080 > log/kubeproxy.log &

m2:
	go run ./cmd/kubelet/kubelet.go --nodename=node2 --nodeip=192.168.1.8 --masterip=http://192.168.1.7:8080 > log/kubelet-m2.log &
	/usr/local/go/bin/go run ./cmd/kube-proxy/kubeproxy.go --masterip=http://192.168.1.7:8080 > log/kubeproxy.log &

stop:
	./scripts/linux/stop.sh

kill:
	sudo docker ps -aq --filter "name=^my-replicaset|^test|^function|^two-con|^sche|^tinyserver|^Gateway|^coreDNS" | xargs -r docker stop
	sudo docker ps -aq --filter "name=^my-replicaset|^test|^function|^two-con|^sche|^tinyserver|^Gateway|^coreDNS" | xargs -r docker rm

testsch:
	./scripts/linux/testsch.sh

delsch:
	./scripts/linux/delsch.sh

testsvc:
	./scripts/linux/testsvc.sh


delsvc:
	./scripts/linux/delsvc.sh

# need project in /home/minik8s, webs in /home/webs
testdns:
	./scripts/linux/testdns.sh

deltestdns:
	./scripts/linux/deldns.sh

testnode:
	/usr/local/go/bin/go run ./cmd/kubectl/kubectl.go get node
