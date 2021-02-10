package main

import (
	"context"
	"driemcoin/main/manifest"
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
			fmt.Println("--------------------------------------------")
			fmt.Println(fmt.Sprintf("FTP2P Version %s.%s.%s-beta", Major, Minor, Patch))
			fmt.Println("--------------------------------------------")
			fmt.Println("")
			alias, _ := cmd.Flags().GetString(flagAlias)
			miner, _ := cmd.Flags().GetString(flagMiner)
			ip, _ := cmd.Flags().GetString(flagIP)
			port, _ := cmd.Flags().GetUint64(flagPort)
			bootstrap := node.NewPeerNode(
				"127.0.0.1",
				8080,
				true,
				manifest.NewAddress("tony"),
				false,
			)
			n := node.NewNode(alias, getDataDirFromCmd(cmd), ip, port,
				manifest.NewAddress(miner), bootstrap)
			err := n.Run(context.Background())
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
	addDefaultRequiredFlags(runCmd)

	runCmd.Flags().String(flagAlias, "user", "Your account alias")
	runCmd.Flags().String(flagMiner, node.DefaultMiner, "miner account of this node to receive block rewards")
	runCmd.Flags().Uint64(flagPort, 8080, "The ip to run the client on")
	runCmd.Flags().String(flagIP, "127.0.0.1", "The ip to run the client with")
	return runCmd
}
