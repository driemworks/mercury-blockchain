package main

import (
	"fmt"
	"ftp2p/manifest"
	"os"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/spf13/cobra"
)

const flagDataDir = "datadir"
const flagIP = "ip"
const flagPort = "port"
const flagMiner = "miner"
const flagName = "name"
const flagKeystoreFile = "keystore"
const flagBootstrapIP = "bootstrap-ip"
const flagBootstrapPort = "bootstrap-port"

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
	ftp2pCmd.AddCommand(walletCmd())

	err := ftp2pCmd.Execute()
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
	return manifest.ExpandPath(dataDir)
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
