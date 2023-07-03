package commands

import (
	"fmt"
	"os"
	"runtime"

	"github.com/DOIDFoundation/node/version"
	"github.com/spf13/cobra"
)

func init() {
	VersionCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show protocol and library versions")
}

// VersionCmd ...
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Version:", version.VersionWithMeta)
		if version.Commit != "" {
			fmt.Println("Git Commit:", version.Commit)
		}
		if version.Date != "" {
			fmt.Println("Git Commit Date:", version.Date)
		}
		fmt.Println("Architecture:", runtime.GOARCH)
		fmt.Println("Go Version:", runtime.Version())
		fmt.Println("Operating System:", runtime.GOOS)
		fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
		fmt.Printf("GOROOT=%s\n", runtime.GOROOT())
	},
}
