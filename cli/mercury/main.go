package main

import (
	"fmt"
	"os"

	"github.com/driemworks/mercury-blockchain/state"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/spf13/cobra"
)

const flagDataDir = "datadir"
const flagIP = "ip"
const flagPort = "port"
const flagMiner = "miner"
const flagName = "name"
const flagKeystoreFile = "keystore"
const flagBootstrap = "bootstrap"
const flagTls = "tls"

func main() {
	var mainCmd = &cobra.Command{
		Use:   "mercury",
		Short: "mercury cli",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
	// TODO these need to be standardized...
	mainCmd.AddCommand(versionCmd)
	mainCmd.AddCommand(runCmd())
	mainCmd.AddCommand(walletCmd())

	err := mainCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func addKeystoreFlag(cmd *cobra.Command) {
	cmd.Flags().String(flagKeystoreFile, "", "Absolute path to the encrypted keystore file")
	cmd.MarkFlagRequired(flagKeystoreFile)
}

func getDataDirFromCmd(cmd *cobra.Command) string {
	dataDir, _ := cmd.Flags().GetString(flagDataDir)
	return state.ExpandPath(dataDir)
}

func addDefaultRequiredFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagDataDir, "", "The absolute path of the node's data dir")
	cmd.MarkFlagRequired(flagDataDir)
}

func incorrectUsageErr() error {
	return fmt.Errorf("incorrect usage")
}

func getPassPhrase(prompt string, confirmation bool) string {
	fmt.Print(prompt)
	var password string
	fmt.Scanln(&password)
	if confirmation {
		fmt.Print("Repeat password: ")
		var confirmationPassword string
		fmt.Scanln(&confirmationPassword)
		if password != confirmationPassword {
			utils.Fatalf("Password should match")
		}
	}
	return password
}
