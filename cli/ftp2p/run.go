package main

import (
	"driemcoin/main/node"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the ftp2p node",
		Run: func(cmd *cobra.Command, args []string) {
			datadir, _ := cmd.Flags().GetString(flagDataDir)
			fmt.Println(fmt.Sprintf("FTP2P Version %s.%s.%s-beta", Major, Minor, Patch))
			fmt.Println("Starting node")
			err := node.Run(datadir)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
	addDefaultRequiredFlags(runCmd)
	return runCmd
}
