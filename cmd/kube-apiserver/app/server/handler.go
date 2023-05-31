package server

import (
	"minik8s/cmd/kube-apiserver/app/apiconfig"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Type         string
	Path         string
	Eventhandler func(c *gin.Context, s *Server)
}

var HandlerTable = []Handler{
	// etcd
	{Type: "GET", Path: apiconfig.PATH, Eventhandler: Get},
	{Type: "PUT", Path: apiconfig.PATH, Eventhandler: Put},
	{Type: "POST", Path: apiconfig.PATH, Eventhandler: Post},
	{Type: "DELETE", Path: apiconfig.PATH, Eventhandler: Delete},
	// node
	{Type: "GET", Path: apiconfig.NODE_PATH, Eventhandler: GetNode},
	{Type: "PUT", Path: apiconfig.NODE_PATH, Eventhandler: AddNode},
	{Type: "DELETE", Path: apiconfig.NODE_PATH, Eventhandler: DeleteNode},
	// pod
	{Type: "GET", Path: apiconfig.POD_PATH, Eventhandler: GetPod},
	{Type: "PUT", Path: apiconfig.POD_PATH, Eventhandler: AddPod},
	{Type: "DELETE", Path: apiconfig.POD_PATH, Eventhandler: DeletePod},
	{Type: "POST", Path: apiconfig.POD_PATH, Eventhandler: UpdatePod},
	// service
	{Type: "GET", Path: apiconfig.SERVICE_PATH, Eventhandler: GetService},
	{Type: "PUT", Path: apiconfig.SERVICE_PATH, Eventhandler: UpdateService},
	{Type: "POST", Path: apiconfig.SERVICE_PATH, Eventhandler: UpdateService},
	{Type: "DELETE", Path: apiconfig.SERVICE_PATH, Eventhandler: DeleteService},
	// dns
	{Type: "GET", Path: apiconfig.DNS_PATH, Eventhandler: GetDNS},
	{Type: "PUT", Path: apiconfig.DNS_PATH, Eventhandler: UpdateDNS},
	{Type: "POST", Path: apiconfig.DNS_PATH, Eventhandler: UpdateDNS},
	{Type: "DELETE", Path: apiconfig.DNS_PATH, Eventhandler: DeleteDNS},
	// ReplicaSet
	{Type: "GET", Path: apiconfig.REPLICASET_PATH, Eventhandler: GetReplicaSet},
	{Type: "PUT", Path: apiconfig.REPLICASET_PATH, Eventhandler: AddReplicaSet},
	{Type: "POST", Path: apiconfig.REPLICASET_PATH, Eventhandler: UpdateReplicaSet},
	{Type: "DELETE", Path: apiconfig.REPLICASET_PATH, Eventhandler: DeleteReplicaSet},
	//job
	{Type: "POST", Path: apiconfig.JOB_PATH, Eventhandler: AddJob},
	{Type: "GET", Path: apiconfig.JOB_PATH, Eventhandler: GetJob},
	{Type: "POST", Path: apiconfig.JOB_FILE_PATH, Eventhandler: AddJobFile},
	{Type: "GET", Path: apiconfig.JOB_FILE_PATH, Eventhandler: GetJobFile},
	//hpa
	{Type: "PUT", Path: apiconfig.HPA_PATH, Eventhandler: AddHPA},
	{Type: "DELETE", Path: apiconfig.HPA_PATH, Eventhandler: DeleteHPA},
	// watch
	{Type: "GET", Path: "/watch/*resource", Eventhandler: Watch}}
