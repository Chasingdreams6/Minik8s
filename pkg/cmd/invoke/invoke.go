package invoke

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"minik8s/cmd/kube-apiserver/app/apiconfig"
	"net/http"
	"strings"
)

type InvokeOptions struct {
	FunctionName string
	Params       string
}

func NewInvokeOptions() *InvokeOptions {
	return &InvokeOptions{
		FunctionName: "",
		Params:       "",
	}
}

func NewCmdInvoke() *cobra.Command {
	o := NewInvokeOptions()

	cmd := &cobra.Command{
		Use:   "invoke",
		Short: "invoke a funtion",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := o.RunInvoke(cmd, args)
			if err != nil {
				return err
			}
			return nil
		},
	}

	//usage := "to use to get the resource"
	cmd.Flags().StringVarP(&o.FunctionName, "name", "f", "", "function name.")
	cmd.Flags().StringVarP(&o.Params, "param", "p", "", "params (in json format)")

	return cmd
}

func (o InvokeOptions) RunInvoke(cmd *cobra.Command, args []string) error {
	prefix := "[kubectl] [invoke] [RunInvoke] "
	if o.FunctionName == "" {
		fmt.Println(prefix, "must have function name.")
		return nil
	}
	fmt.Println("[kubectl] [invoke] [RunInvoke] function name: ", o.FunctionName, " params: ", o.Params)

	req, err := http.NewRequest("POST", apiconfig.Server_URL+"/invoke/"+o.FunctionName, strings.NewReader(o.Params))
	if err != nil {
		fmt.Println(prefix, "Error creating request:", err)
		return err
	}

	// 设置请求头的 Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(prefix, "Error sending request:", err)
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println("Final respond:", string(body))

	return nil
}
