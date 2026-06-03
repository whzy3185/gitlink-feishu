package config

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	internalConfig "github.com/gitlink-org/gitlink-cli/internal/config"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
)

func NewConfigCmd(translators ...*i18n.Translator) *cobra.Command {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	cmd := &cobra.Command{
		Use:   "config",
		Short: tr.T("cmd.config.short"),
	}
	cmd.AddCommand(newInitCmd(tr))
	cmd.AddCommand(newSetCmd(tr))
	cmd.AddCommand(newGetCmd(tr))
	cmd.AddCommand(newListCmd(tr))
	return cmd
}

func newInitCmd(translators ...*i18n.Translator) *cobra.Command {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	return &cobra.Command{
		Use:   "init",
		Short: tr.T("cmd.config.init.short"),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := internalConfig.DefaultConfig()
			if err := internalConfig.Save(cfg); err != nil {
				return errors.New(tr.Tf("error.config.save_failed", i18n.Args{"message": err.Error()}))
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), tr.Tf("success.config.initialized", i18n.Args{"path": internalConfig.ConfigPath()}))
			return err
		},
	}
}

func newSetCmd(translators ...*i18n.Translator) *cobra.Command {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: tr.T("cmd.config.set.short"),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := internalConfig.Set(args[0], args[1]); err != nil {
				return err
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), tr.Tf("success.config.set", i18n.Args{
				"key":   args[0],
				"value": args[1],
			}))
			return err
		},
	}
}

func newGetCmd(translators ...*i18n.Translator) *cobra.Command {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	return &cobra.Command{
		Use:   "get <key>",
		Short: tr.T("cmd.config.get.short"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			val, err := internalConfig.Get(args[0])
			if err != nil {
				return err
			}
			if val == "" {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", args[0], tr.T("output.config.not_set"))
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", args[0], val)
			return err
		},
	}
}

func newListCmd(translators ...*i18n.Translator) *cobra.Command {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	return &cobra.Command{
		Use:   "list",
		Short: tr.T("cmd.config.list.short"),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internalConfig.Load()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if _, err := fmt.Fprintf(out, "base_url:       %s\n", cfg.BaseURL); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "default_format: %s\n", cfg.Format); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "editor:         %s\n", cfg.Editor); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "pager:          %s\n", cfg.Pager); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "lang:           %s\n", cfg.Lang); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, tr.Tf("output.config.file", i18n.Args{"path": internalConfig.ConfigPath()}))
			return err
		},
	}
}
