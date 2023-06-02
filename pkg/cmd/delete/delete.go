package delete

import (
	"fmt"
	"minik8s/cmd/kube-apiserver/app/apiconfig"
	"minik8s/pkg/util/web"
	"net/url"

	"github.com/spf13/cobra"
)

// DeleteOptions is the commandline options for 'delete' sub command
type DeleteOptions struct {
	DeleteAll bool
}

// NewDeleteOptions returns an initialized DeleteOptions instance
func NewDeleteOptions() *DeleteOptions {
	return &DeleteOptions{
		DeleteAll: false,
	}
}

// NewCmdDelete returns new initialized instance of Delete sub command
func NewCmdDelete() *cobra.Command {
	o := NewDeleteOptions()

	cmd := &cobra.Command{
		Use:   "delete TYPE [NAME | -all]",
		Short: "Delete a resource from stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := o.RunDelete(cmd, args)
			if err != nil {
				return err
			}
			return nil
		},
	}

	//usage := "to use to delete the resource"
	cmd.Flags().BoolVarP(&o.DeleteAll, "all", "a", false, "delete all.")

	return cmd
}

// RunDelete performs the creation
func (o *DeleteOptions) RunDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		fmt.Println("[kubectl] [delete] [RunDelete] must have TYPE.")
		return nil
	}

	switch args[0] {
	case "pod":
		{
			return o.RunDeletePod(cmd, args)
		}
	case "replicaset":
		{
			return o.RunDeleteReplicaSet(cmd, args)
		}
	case "service":
		{
			return o.RunDeleteService(cmd, args)
		}
	case "hpa":
		{
			return o.RunDeleteHPA(cmd, args)
		}
	case "dns":
		{
			return o.RunDeleteDNS(cmd, args)
		}
	case "job":
		{
			return o.RunDeleteJob(cmd, args)
		}
	case "function":
		{
			return o.RunDeleteFunction(cmd, args)
		}
	default:
		{
			fmt.Printf("[kubectl] [delete] [RunDelete] %s is not supported.\n", args[0])
			return nil
		}
	}

}

func (o *DeleteOptions) RunDeletePod(cmd *cobra.Command, args []string) error {
	prefix := "[kubectl] [delete] [RunDeletePod] "
	values := url.Values{}
	if o.DeleteAll {
		values.Add("all", "true")
	}
	if len(args) > 1 {
		values.Add("PodName", args[1])
		//body = bytes.NewBuffer([]byte(args[1]))
	}

	err := web.SendHttpRequest("DELETE", apiconfig.Server_URL+apiconfig.POD_PATH+"?"+values.Encode(),
		web.WithPrefix(prefix),
		web.WithLog(true))
	if err != nil {
		return err
	}

	fmt.Println("pod Delete successfully")
	return nil
}

func (o *DeleteOptions) RunDeleteReplicaSet(cmd *cobra.Command, args []string) error {
	prefix := "[kubectl] [delete] [RunDelete] "
	values := url.Values{}
	if o.DeleteAll {
		values.Add("all", "true")
	}
	if len(args) > 1 {
		values.Add("Name", args[1])
		//body = bytes.NewBuffer([]byte(args[1]))
	}

	err := web.SendHttpRequest("DELETE", apiconfig.Server_URL+apiconfig.REPLICASET_PATH+"?"+values.Encode(),
		web.WithPrefix(prefix),
		web.WithLog(true))
	if err != nil {
		return err
	}

	fmt.Println("Delete ReplicaSet successfully")
	return nil
}

func (o *DeleteOptions) RunDeleteService(cmd *cobra.Command, args []string) error {
	values := url.Values{}
	if o.DeleteAll {
		values.Add("all", "true")
	}
	if len(args) > 1 {
		values.Add("ServiceName", args[1])
		//body = bytes.NewBuffer([]byte(args[1]))
	}

	err := web.SendHttpRequest("DELETE", apiconfig.Server_URL+apiconfig.SERVICE_PATH+"?"+values.Encode(),
		web.WithPrefix("[kubectl] [delete] [DeleteService] "),
		web.WithLog(true))
	if err != nil {
		return err
	}

	fmt.Println("[kubectl] send service Delete request successfully")
	return nil
}

func (o *DeleteOptions) RunDeleteDNS(cmd *cobra.Command, args []string) error {
	// 创建 PUT 请求
	if len(args) < 1 || args[0] != "dns" {
		fmt.Println("only support dns.")
		return nil
	}

	values := url.Values{}
	if o.DeleteAll {
		values.Add("all", "true")
	}
	if len(args) > 1 {
		values.Add("DnsConfigName", args[1])
		//body = bytes.NewBuffer([]byte(args[1]))
	}

	err := web.SendHttpRequest("DELETE", apiconfig.Server_URL+apiconfig.DNS_PATH+"?"+values.Encode(),
		web.WithPrefix("[kubectl] [delete] [DeleteDNS] "),
		web.WithLog(true))
	if err != nil {
		return err
	}

	fmt.Println("[kubectl] send DNS Delete request successfully")
	return nil
}

func (o *DeleteOptions) RunDeleteHPA(cmd *cobra.Command, args []string) error {
	values := url.Values{}
	if o.DeleteAll {
		values.Add("all", "true")
	}
	if len(args) > 1 {
		values.Add("HPAName", args[1])
		//body = bytes.NewBuffer([]byte(args[1]))
	}

	err := web.SendHttpRequest("DELETE", apiconfig.Server_URL+apiconfig.HPA_PATH+"?"+values.Encode(),
		web.WithPrefix("[kubectl] [delete] [DeleteHPA] "),
		web.WithLog(true))
	if err != nil {
		return err
	}

	fmt.Println("hpa Delete successfully")
	return nil
}

func (o *DeleteOptions) RunDeleteJob(cmd *cobra.Command, args []string) error {
	prefix := "[kubectl] [delete] [RunDeleteJob] "
	values := url.Values{}
	if o.DeleteAll {
		values.Add("all", "true")
	}
	if len(args) > 1 {
		values.Add("JobName", args[1])
		//body = bytes.NewBuffer([]byte(args[1]))
	}

	err := web.SendHttpRequest("DELETE", apiconfig.Server_URL+apiconfig.JOB_PATH+"?"+values.Encode(),
		web.WithPrefix(prefix),
		web.WithLog(true))
	if err != nil {
		return err
	}

	fmt.Println("job Delete successfully")
	return nil
}

func (o *DeleteOptions) RunDeleteFunction(cmd *cobra.Command, args []string) error {
	prefix := "[kubectl] [delete] [DeleteHPA] "
	values := url.Values{}
	if o.DeleteAll {
		values.Add("all", "true")
	}
	if len(args) > 1 {
		values.Add("FunctionName", args[1])
		//body = bytes.NewBuffer([]byte(args[1]))
	}

	err := web.SendHttpRequest("DELETE", apiconfig.Server_URL+apiconfig.FUNCTION_PATH+"?"+values.Encode(),
		web.WithPrefix(prefix),
		web.WithLog(true))
	if err != nil {
		return err
	}

	fmt.Println("function Delete successfully")
	return nil
}
