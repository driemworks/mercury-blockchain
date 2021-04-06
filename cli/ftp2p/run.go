package main

import (
	"context"
	"fmt"
	"ftp2p/common"
	"ftp2p/node"
	"ftp2p/state"
	"ftp2p/wallet"
	"math/rand"
	"os"

	"github.com/raphamorim/go-rainbow"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the ftp2p node",
		Run: func(cmd *cobra.Command, args []string) {

			name, _ := cmd.Flags().GetString(flagName)
			miner, _ := cmd.Flags().GetString(flagMiner)
			ip, _ := cmd.Flags().GetString(flagIP)
			port, _ := cmd.Flags().GetUint64(flagPort)
			bootstrapIP, _ := cmd.Flags().GetString(flagBootstrapIP)
			bootstrapPort, _ := cmd.Flags().GetUint64(flagBootstrapPort)

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
			fmt.Printf("\t\t Using bootstrap node: %s:%s\n", rainbow.Green(bootstrapIP), rainbow.Green(fmt.Sprint(bootstrapPort)))
			fmt.Println("")
			bootstrap := common.NewPeerNode(
				"bootstrap",
				bootstrapIP,
				bootstrapPort,
				true,
				state.NewAddress(""), // 0x96131b31b9935f6388502b502cf544c1a8c65ad6
				false,
			)
			// decrypt encryption public key
			keys, err := wallet.LoadEncryptionKeys(getDataDirFromCmd(cmd), password)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			publicKey := keys[:32]
			n := node.NewNode(name, getDataDirFromCmd(cmd), ip, port,
				state.NewAddress(miner), string(publicKey),
				bootstrap)
			err = n.Run(context.Background())
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	addDefaultRequiredFlags(runCmd)

	runCmd.Flags().String(flagName, fmt.Sprintf("user-%d", rand.Int()), "Your username")
	runCmd.Flags().String(flagMiner, node.DefaultMiner, "miner account of this node to receive block rewards")
	runCmd.Flags().Uint64(flagPort, 8080, "The ip to run the client on")
	runCmd.Flags().String(flagIP, "127.0.0.1", "The ip to run the client with")
	runCmd.Flags().String(flagBootstrapIP, "127.0.0.1", "default bootstrap server to interconnect peers")
	runCmd.Flags().Uint64(flagBootstrapPort, 8080, "default bootstrap server port to interconnect peers")
	return runCmd
}
