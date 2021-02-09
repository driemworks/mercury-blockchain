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
			fmt.Println(fmt.Sprintf("FTP2P Version %s.%s.%s-beta", Major, Minor, Patch))

			ip, _ := cmd.Flags().GetString(flagIP)
			port, _ := cmd.Flags().GetUint64(flagPort)
			fmt.Printf("Listening on port %d\n", port)

			bootstrap := node.NewPeerNode(
				"127.0.0.1",
				8080,
				true,
				false,
			)
			n := node.NewNode(getDataDirFromCmd(cmd), ip, port, bootstrap)
			err := n.Run()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
	addDefaultRequiredFlags(runCmd)
	runCmd.Flags().Uint64(flagPort, 8080, "The ip to run the client on")
	runCmd.Flags().String(flagIP, "127.0.0.1", "The ip to run the client with")
	return runCmd
}
