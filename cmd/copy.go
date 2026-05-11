package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/config"
	"github.com/thereisnotime/sshroute/internal/network"
	"github.com/thereisnotime/sshroute/internal/ssh"
)

// RealSCP is the absolute path to the system scp binary.
const RealSCP = "/usr/bin/scp"

var copyCmd = &cobra.Command{
	Use:   "copy <alias> <src> <dst>",
	Short: "Copy files to/from a configured host using resolved SSH parameters",
	Long: `Copy files to or from a configured host using scp with the same
SSH parameters (key, port, jump) that sshroute would use for the active network.

Use <alias>:<path> syntax for remote paths, same as scp:

  sshroute copy myserver ./local.txt myserver:/remote/path/
  sshroute copy myserver myserver:/remote/file.txt ./local/`,
	Args:              cobra.ExactArgs(3),
	RunE:              runCopy,
	ValidArgsFunction: completeAliases,
}

func runCopy(cmd *cobra.Command, args []string) error {
	alias, src, dst := args[0], args[1], args[2]

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if _, ok := cfg.Hosts[alias]; !ok {
		return fmt.Errorf("host %q not found in config", alias)
	}

	detectedNetwork, err := network.Detect(cfg.Networks)
	if err != nil {
		return fmt.Errorf("detecting network: %w", err)
	}
	slog.Debug("copy: detected network", "network", detectedNetwork)

	params, err := ssh.Resolve(cfg, alias, detectedNetwork)
	if err != nil {
		return fmt.Errorf("resolving params: %w", err)
	}

	scpBin := resolveSCPBinary()
	argv := buildSCPArgv(scpBin, params, src, dst, alias)

	if dryRun {
		fmt.Println("[dry-run] " + strings.Join(argv, " "))
		return nil
	}

	slog.Debug("copy: executing", "argv", argv)
	c := exec.Command(argv[0], argv[1:]...) // #nosec G204 -- argv[0] is always the resolved scp binary
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("scp: %w", err)
	}
	return nil
}

// buildSCPArgv constructs the scp command from resolved SSH params.
// Remote paths are identified by the alias: prefix and rewritten to user@host:path.
func buildSCPArgv(scpBin string, params config.SSHParams, src, dst, alias string) []string {
	argv := []string{scpBin}

	if params.Port != 0 {
		argv = append(argv, "-P", fmt.Sprintf("%d", params.Port))
	}
	if params.Key != "" {
		argv = append(argv, "-i", params.Key)
	}
	if params.Jump != "" {
		argv = append(argv, "-J", params.Jump)
	}

	remote := params.Host
	if params.User != "" {
		remote = params.User + "@" + params.Host
	}

	argv = append(argv, rewriteRemote(src, alias, remote), rewriteRemote(dst, alias, remote))
	return argv
}

// rewriteRemote replaces "<alias>:<path>" with "<user@host>:<path>".
func rewriteRemote(arg, alias, remote string) string {
	prefix := alias + ":"
	if strings.HasPrefix(arg, prefix) {
		return remote + ":" + arg[len(prefix):]
	}
	return arg
}

// resolveSCPBinary returns the scp binary path, checking SSHROUTE_SCP env var first.
func resolveSCPBinary() string {
	if v := os.Getenv("SSHROUTE_SCP"); v != "" {
		return v
	}
	if path, err := exec.LookPath("scp"); err == nil {
		return path
	}
	return RealSCP
}

func init() {
	rootCmd.AddCommand(copyCmd)
}
