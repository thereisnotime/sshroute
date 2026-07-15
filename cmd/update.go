package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/update"
	"github.com/thereisnotime/sshroute/internal/version"
)

var (
	updateCheck bool
	updateForce bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update sshroute to the latest release",
	Long: "Download the latest GitHub release for this platform, verify its sha256 against " +
		"checksums.txt (and the cosign signature when cosign is installed), then replace the " +
		"running binary in place. Installed via a package manager or `go install`? Update with that instead.",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := update.Run(context.Background(), version.Version, update.Options{
			CheckOnly: updateCheck,
			Force:     updateForce,
		})
		return err
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&updateCheck, "check", false, "only report whether a newer version is available")
	updateCmd.Flags().BoolVar(&updateForce, "force", false, "reinstall even if already on the latest version")
}
