package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const flagDataDir = "datadir"

func main() {
	var ftp2pCmd = &cobra.Command{
		Use:   "ftp2p",
		Short: "ftp2p cli",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
	// TODO these need to be standardized...
	ftp2pCmd.AddCommand(versionCmd)
	ftp2pCmd.AddCommand(runCmd())

	err := ftp2pCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func addDefaultRequiredFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagDataDir, "", "The absolute path of the node's data dir")
	cmd.MarkFlagRequired(flagDataDir)
}

func incorrectUsageErr() error {
	return fmt.Errorf("incorrect usage")
}
