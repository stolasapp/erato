package command

import (
	"bytes"
	"errors"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/stolasapp/erato/internal/sec"
	"github.com/stolasapp/erato/internal/storage/db"
)

func userCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "User commands",
	}
	cmd.AddCommand(
		userCreateCommand(),
		userDeleteCommand(),
	)
	return cmd
}

func userCreateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "create NAME",
		Short: "Create user",
		Long: "Creates user entry for the provided username and password. Passwords may be\n" +
			"provided via stdin or through the interactive prompt.",

		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (runErr error) {
			_, logger, store, err := loadConfig(cmd.Context())
			if err != nil {
				return err
			}
			defer func() {
				if err := store.Close(); err != nil {
					runErr = errors.Join(runErr, err)
				}
			}()

			name := args[0]
			if passwd, err := prompt("password: ", true); err != nil {
				return err
			} else if hash, err := sec.HashPassword(passwd); err != nil {
				return err
			} else if err = store.UpsertUser(cmd.Context(), db.User{
				Name:         name,
				PasswordHash: hash,
			}); err != nil {
				return err
			}

			logger.InfoContext(cmd.Context(), "created user", slog.String("name", name))
			return nil
		},
	}
}

func userDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete user",
		Long: "Permanently deletes the user and all associated data. " +
			"This operation is permanent and irreversible.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (runErr error) {
			_, logger, store, err := loadConfig(cmd.Context())
			if err != nil {
				return err
			}
			defer func() {
				if err := store.Close(); err != nil {
					runErr = errors.Join(runErr, err)
				}
			}()

			name := args[0]
			logger = logger.With(slog.String("name", name))
			user, err := store.GetUserByName(cmd.Context(), name)
			if err != nil {
				return err
			}
			resp, err := prompt("Are you sure you want to delete this user? [y|N] ", false)
			if !bytes.Equal(resp, []byte{'y'}) || err != nil {
				logger.InfoContext(cmd.Context(), "aborted user deletion")
				return err
			}
			if err = store.DeleteUser(cmd.Context(), user.ID); err != nil {
				return err
			}
			logger.InfoContext(cmd.Context(), "user deleted")
			return nil
		},
	}
}
