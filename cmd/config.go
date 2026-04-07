package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/thereisnotime/sshroute/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show config file path or open it in $EDITOR",
	RunE:  runConfig,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open the config file in $EDITOR",
	RunE:  runConfigEdit,
}

func runConfig(cmd *cobra.Command, args []string) error {
	path := cfgFile
	if path == "" {
		var err error
		path, err = config.DefaultConfigPath()
		if err != nil {
			return fmt.Errorf("resolving config path: %w", err)
		}
	}
	fmt.Println(path)
	return nil
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	path := cfgFile
	if path == "" {
		var err error
		path, err = config.DefaultConfigPath()
		if err != nil {
			return fmt.Errorf("resolving config path: %w", err)
		}
	}

	// Ensure the config file and its parent directory exist before opening.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory %q: %w", dir, err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("creating config file %q: %w", path, err)
		}
		f.Close()
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}

	editorPath, err := exec.LookPath(editor)
	if err != nil {
		return fmt.Errorf("editor %q not found in PATH: %w", editor, err)
	}

	if err := syscall.Exec(editorPath, []string{editorPath, path}, os.Environ()); err != nil {
		return fmt.Errorf("exec %s: %w", editorPath, err)
	}
	// Unreachable — syscall.Exec replaces the process.
	return nil
}

func init() {
	configCmd.AddCommand(configEditCmd)
	rootCmd.AddCommand(configCmd)
}
