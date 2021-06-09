package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/driemworks/mercury-blockchain/node"

	"github.com/raphamorim/go-rainbow"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	// forces logrus to use colors even in windows (https://github.com/sirupsen/logrus/issues/306)
	var testf = new(log.TextFormatter)
	testf.ForceColors = true
	log.SetFormatter(testf)
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the mercury node",
		Run: func(cmd *cobra.Command, args []string) {
			name, _ := cmd.Flags().GetString(flagName)
			address, _ := cmd.Flags().GetString(flagAddress)
			host, _ := cmd.Flags().GetString(flagHost)
			port, _ := cmd.Flags().GetUint64(flagPort)
			rpcHost, _ := cmd.Flags().GetString(flatRPCHost)
			rpcPort, _ := cmd.Flags().GetUint64(flagRPCPort)
			bootstrap, _ := cmd.Flags().GetString(flagBootstrap)
			stake, _ := cmd.Flags().GetInt("stake")
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
			log.Infoln("Starting mercury")
			log.Infoln(fmt.Sprintf("Version %s.%s.%s-beta\n", Major, Minor, Patch))
			n := node.NewNode(name, getDataDirFromCmd(cmd), address, rpcHost, port, false, stake)
			err := n.Run(context.Background(), host, int(port), rpcHost, rpcPort,
				bootstrap, name)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	addDefaultRequiredFlags(runCmd)

	runCmd.Flags().String(flagName, fmt.Sprintf("user-%d", rand.Int()), "Your username")
	runCmd.Flags().String(flagAddress, "", "miner account of this node to receive block rewards")
	runCmd.Flags().Uint64(flagPort, 8080, "The port to run the p2p client on")
	runCmd.Flags().Uint64(flagRPCPort, 9080, "The port to run the rpc server on")
	runCmd.Flags().String(flagHost, "127.0.0.1", "The host to run the client with")
	runCmd.Flags().String(flatRPCHost, "0.0.0.0", "The host to run the rpc server on")
	runCmd.Flags().String(flagBootstrap, "", "the bootstrap server to interconnect peers")
	runCmd.Flags().Bool(flagTls, false, "true if tls is enabled (for the rpc server), false otherwise")
	return runCmd
}
