package cmd

import (
	"github.com/spf13/cobra"
	"minik8s/pkg/cmd/apply"
	"minik8s/pkg/cmd/create"
	"minik8s/pkg/cmd/delete"
	"minik8s/pkg/cmd/get"
	"minik8s/pkg/cmd/invoke"
)

// NewKubectlCommand creates the `kubectl` command and its nested children.
func NewKubectlCommand() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "kubectl [command] [flag]",
		Short: "kubectl controls the Kubernetes cluster manager",
	}
	rootCmd.AddCommand(create.NewCmdCreate())
	rootCmd.AddCommand(delete.NewCmdDelete())
	rootCmd.AddCommand(get.NewCmdGet())
	rootCmd.AddCommand(apply.NewCmdApply())
	rootCmd.AddCommand(invoke.NewCmdInvoke())
	return rootCmd
}
