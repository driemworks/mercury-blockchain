package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/driemworks/mercury-blockchain/node"

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
			bootstrap, _ := cmd.Flags().GetString(flagBootstrap)
			// validate/fix data issues (thanks Windows)
			if strings.Contains(bootstrap, "C:/Program Files/Git") {
				bootstrap = strings.Split(bootstrap, "C:/Program Files/Git")[1]
			}
			fmt.Println("\n\t\t" + rainbow.Bold(rainbow.Hex("#B164E3", `
       __  ___                         
      /  |/  /__ __________ ________ __
     / /|_/ / -_) __/ __/ // / __/ // /
    /_/  /_/\__/_/  \__/\_,_/_/  \_, / 
	    		        /___/  
			`)))
			fmt.Println(fmt.Sprintf("\tVersion %s.%s.%s-beta\n", Major, Minor, Patch))
			n := node.NewNode(name, getDataDirFromCmd(cmd), miner, "127.0.0.1", port, false)
			err := n.Run(context.Background(), ip, int(port), bootstrap, name)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	addDefaultRequiredFlags(runCmd)

	runCmd.Flags().String(flagName, fmt.Sprintf("user-%d", rand.Int()), "Your username")
	runCmd.Flags().String(flagMiner, "", "miner account of this node to receive block rewards")
	runCmd.Flags().Uint64(flagPort, 8081, "The port to run the client on")
	runCmd.Flags().String(flagIP, "127.0.0.1", "The ip to run the client with")
	runCmd.Flags().String(flagBootstrap, "", "the bootstrap server to interconnect peers")
	runCmd.Flags().Bool(flagTls, false, "true if tls is enabled (for the rpc server), false otherwise")
	return runCmd
}
