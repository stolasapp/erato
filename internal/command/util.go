package command

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"

	"golang.org/x/term"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
	"github.com/stolasapp/erato/internal/storage"
)

type configKey struct{}

func prompt(prompt string, mask bool) ([]byte, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		if _, err := os.Stderr.WriteString(prompt); err != nil {
			return nil, err
		}
	}
	return readLine(os.Stdin, mask)
}

// cloned from term.readPasswordLine.
func readLine(stdin *os.File, mask bool) ([]byte, error) {
	if mask && term.IsTerminal(int(stdin.Fd())) {
		return term.ReadPassword(int(stdin.Fd()))
	}
	var buf [1]byte
	var ret []byte

	for {
		n, err := stdin.Read(buf[:])
		if n > 0 {
			switch buf[0] {
			case '\b':
				if len(ret) > 0 {
					ret = ret[:len(ret)-1]
				}
			case '\n':
				if runtime.GOOS != "windows" {
					return ret, nil
				}
				// otherwise ignore \n
			case '\r':
				if runtime.GOOS == "windows" {
					return ret, nil
				}
				// otherwise ignore \r
			default:
				ret = append(ret, buf[0]) //nolint:gosec // erroneous error
			}
			continue
		}
		if err != nil {
			if errors.Is(err, io.EOF) && len(ret) > 0 {
				return ret, nil
			}
			return ret, err
		}
	}
}

func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown-dev"
	}
	ver := "unknown"
	dirty := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			ver = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}
	if dirty {
		ver += "-dev"
	}
	return ver
}

func loadConfig(ctx context.Context) (*eratov1.Config, *slog.Logger, storage.Store, error) {
	cfg, ok := ctx.Value(configKey{}).(*eratov1.Config)
	if !ok {
		return nil, nil, nil, errors.New("config file resolution failed")
	}
	logger := slog.Default()
	store, err := storage.NewDB(ctx, cfg, logger)
	if err != nil {
		return nil, nil, nil, err
	}

	return cfg, logger, store, nil
}
