package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Major = "1"
const Minor = "0"
const Patch = "1"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(fmt.Sprintf("Version %s.%s.%s-beta", Major, Minor, Patch))
	},
}
