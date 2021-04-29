package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"

	"github.com/driemworks/mercury-blockchain/core"
	"github.com/driemworks/mercury-blockchain/node"
	"github.com/driemworks/mercury-blockchain/state"

	"github.com/raphamorim/go-rainbow"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the mercury node",
		Run: func(cmd *cobra.Command, args []string) {

			name, _ := cmd.Flags().GetString(flagName)
			miner, _ := cmd.Flags().GetString(flagMiner)
			ip, _ := cmd.Flags().GetString(flagIP)
			port, _ := cmd.Flags().GetUint64(flagPort)
			bootstrapIP, _ := cmd.Flags().GetString(flagBootstrapIP)
			bootstrapPort, _ := cmd.Flags().GetUint64(flagBootstrapPort)
			tls, _ := cmd.Flags().GetBool(flagTls)
			// TODO: How do I hide this?
			password := getPassPhrase("Password: ", false)

			fmt.Println("")
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
			fmt.Println(fmt.Sprintf("\t\t Version %s.%s.%s-beta", Major, Minor, Patch))
			fmt.Printf("\t\t Using address: %s\n", rainbow.Green(miner))
			fmt.Println("")
			bootstrap := core.NewPeerNode(
				"bootstrap",
				bootstrapIP,
				bootstrapPort,
				true,
				state.NewAddress(""),
				false,
			)
			n := node.NewNode(name, getDataDirFromCmd(cmd), ip, port,
				state.NewAddress(miner), bootstrap, password, tls)
			err := n.Run_RPC(context.Background())
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	addDefaultRequiredFlags(runCmd)

	runCmd.Flags().String(flagName, fmt.Sprintf("user-%d", rand.Int()), "Your username")
	runCmd.Flags().String(flagMiner, node.DefaultMiner, "miner account of this node to receive block rewards")
	runCmd.Flags().Uint64(flagPort, 8081, "The ip to run the client on")
	runCmd.Flags().String(flagIP, "127.0.0.1", "The ip to run the client with")
	runCmd.Flags().String(flagBootstrapIP, "127.0.0.1", "the bootstrap server to interconnect peers")
	runCmd.Flags().Uint64(flagBootstrapPort, 8081, "the bootstrap server port to interconnect peers")
	runCmd.Flags().Bool(flagTls, false, "true if tls is enabled, false otherwise")
	return runCmd
}
