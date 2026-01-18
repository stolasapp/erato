// Package config handles resolving configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"buf.build/go/protovalidate"
	"buf.build/go/protoyaml"
	"github.com/adrg/xdg"
	"google.golang.org/protobuf/proto"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
)

// Default returns a version of the config with all default values populated.
// Note that this configuration is _not_ valid, as the user must set root_uri.
func Default() *eratov1.Config {
	return eratov1.Config_builder{
		LogLevel:   eratov1.Config_INFO,
		RpcAddress: proto.String("localhost:9998"),
		WebAddress: proto.String("localhost:9999"),
		DbFilepath: filepath.Join(xdg.DataHome, "erato", "db.sqlite"),
		RootUri:    "", // must be set by the user
		DevMode:    false,
	}.Build()
}

// Load loads a YAML configuration file from a path, merges it with defaults, and
// validates it for completeness.
func Load(path string) (*eratov1.Config, error) {
	bytes, err := os.ReadFile(path) //nolint:gosec // allow the config file to be loaded from anywhere
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	fileConfig := &eratov1.Config{}
	if err = (protoyaml.UnmarshalOptions{Path: path}).Unmarshal(bytes, fileConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file at %s: %w", path, err)
	}
	cfg := Default()
	proto.Merge(cfg, fileConfig)
	if err = protovalidate.Validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	return cfg, nil
}
