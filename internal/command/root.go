// Package command contains the CLI command constructors.
package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"buf.build/go/protoyaml"
	"github.com/adrg/xdg"
	"github.com/spf13/cobra"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"

	"github.com/stolasapp/erato/internal/config"
	"github.com/stolasapp/erato/internal/observability"
)

// RootCommand instantiates the root command, with all sub-commands bound.
func RootCommand() *cobra.Command {
	configFilePath := filepath.Join(xdg.ConfigHome, "erato.yaml")
	cmd := &cobra.Command{
		Use:          "erato [command] [flags]",
		Short:        "The archive proxy and tracking tool",
		Version:      version(),
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) (err error) {
			cfg, err := loadOrInitConfig(configFilePath)
			if err != nil {
				return fmt.Errorf("failed to load configuration file: %w", err)
			}
			logger := observability.InitSlog(cfg)
			logger.DebugContext(cmd.Context(), "configuration loaded", slog.Any("config", cfg))
			slog.SetDefault(logger)
			cmd.SetContext(context.WithValue(cmd.Context(), configKey{}, cfg))
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(
		&configFilePath,
		"config", "c",
		configFilePath,
		"path to the configuration file",
	)

	cmd.AddCommand(
		serveCommand(),
		userCommand(),
	)

	return cmd
}

func loadOrInitConfig(configFilePath string) (*eratov1.Config, error) {
	cfg, err := config.Load(configFilePath)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return cfg, err
	}

	resp, initErr := prompt(fmt.Sprintf("Config not found at %s. Create one? [y|N] ", configFilePath), false)
	if initErr != nil || !bytes.Equal(resp, []byte("y")) {
		return nil, errors.Join(err, initErr)
	}

	resp, err = prompt("Enter the root upstream URL for the archive: ", false)
	if err != nil {
		return nil, err
	}

	cfg = config.Default()
	cfg.SetRootUri(string(resp))
	data, err := protoyaml.MarshalOptions{}.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config message to YAML: %w", err)
	}
	if err = os.WriteFile(configFilePath, data, 0600); err != nil { //nolint:mnd // owner rw access
		return nil, fmt.Errorf("failed to write config file to %s: %w", configFilePath, err)
	}
	return cfg, nil
}
