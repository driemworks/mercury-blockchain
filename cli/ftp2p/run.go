package main

import (
	"context"
	"driemcoin/main/manifest"
	"driemcoin/main/node"
	"fmt"
	"os"

	"github.com/raphamorim/go-rainbow"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the ftp2p node",
		Run: func(cmd *cobra.Command, args []string) {

			alias, _ := cmd.Flags().GetString(flagAlias)
			miner, _ := cmd.Flags().GetString(flagMiner)
			ip, _ := cmd.Flags().GetString(flagIP)
			port, _ := cmd.Flags().GetUint64(flagPort)

			fmt.Println("")
			fmt.Println("\t\t " + rainbow.Bold(rainbow.Hex("#B164E3", `/$$$$$$$$ /$$$$$$$$ /$$$$$$$   /$$$$$$  /$$$$$$$ 
		| $$_____/|__  $$__/| $$__  $$ /$$__  $$| $$__  $$
		| $$         | $$   | $$  \ $$|__/  \ $$| $$  \ $$
		| $$$$$      | $$   | $$$$$$$/  /$$$$$$/| $$$$$$$/
		| $$__/      | $$   | $$____/  /$$____/ | $$____/ 
		| $$         | $$   | $$      | $$      | $$      
		| $$         | $$   | $$      | $$$$$$$$| $$      
		|__/         |__/   |__/      |________/|__/      `)))
			fmt.Println("")
			fmt.Println(fmt.Sprintf("\t\t Version %s.%s.%s-beta, driemworks", Major, Minor, Patch))
			fmt.Printf("\t\t Using miner address: %s\n", miner)
			fmt.Printf("\t\t Using bootstrap node: %s:%d\n", "127.0.0.1", 8080)
			fmt.Println("")
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
