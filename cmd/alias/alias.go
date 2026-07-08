package alias

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	internalConfig "github.com/gitlink-org/gitlink-cli/internal/config"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
)

func NewAliasCmd(translators ...*i18n.Translator) *cobra.Command {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	cmd := &cobra.Command{
		Use:   "alias",
		Short: tr.T("cmd.alias.short"),
	}
	cmd.AddCommand(newSetCmd(tr))
	cmd.AddCommand(newListCmd(tr))
	cmd.AddCommand(newDeleteCmd(tr))
	cmd.AddCommand(newImportCmd(tr))
	return cmd
}

func newSetCmd(translators ...*i18n.Translator) *cobra.Command {
	tr := resolve(translators)
	cmd := &cobra.Command{
		Use:   "set <name> <expansion>...",
		Short: tr.T("cmd.alias.set.short"),
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			// Everything after the name forms the expansion, so quoting on the
			// shell is optional for multi-word expansions.
			expansion := strings.Join(args[1:], " ")
			if strings.TrimSpace(expansion) == "" {
				return errors.New(tr.T("error.alias.empty_expansion"))
			}

			cfg, err := internalConfig.Load()
			if err != nil {
				return err
			}
			if cfg.Aliases == nil {
				cfg.Aliases = map[string]string{}
			}
			cfg.Aliases[name] = expansion
			if err := internalConfig.Save(cfg); err != nil {
				return errors.New(tr.Tf("error.alias.save_failed", i18n.Args{"message": err.Error()}))
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), tr.Tf("success.alias.set", i18n.Args{
				"name":      name,
				"expansion": expansion,
			}))
			return err
		},
	}
	// Once the alias name is read, treat the rest of the line as the literal
	// expansion so flag-looking tokens (e.g. --label) are not parsed here.
	cmd.Flags().SetInterspersed(false)
	return cmd
}

func newListCmd(translators ...*i18n.Translator) *cobra.Command {
	tr := resolve(translators)
	return &cobra.Command{
		Use:   "list",
		Short: tr.T("cmd.alias.list.short"),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internalConfig.Load()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(cfg.Aliases) == 0 {
				_, err := fmt.Fprintln(out, tr.T("output.alias.none"))
				return err
			}
			names := make([]string, 0, len(cfg.Aliases))
			for name := range cfg.Aliases {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				if _, err := fmt.Fprintf(out, "%s: %s\n", name, cfg.Aliases[name]); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func newDeleteCmd(translators ...*i18n.Translator) *cobra.Command {
	tr := resolve(translators)
	return &cobra.Command{
		Use:   "delete <name>",
		Short: tr.T("cmd.alias.delete.short"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := internalConfig.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Aliases[name]; !ok {
				return errors.New(tr.Tf("error.alias.not_found", i18n.Args{"name": name}))
			}
			delete(cfg.Aliases, name)
			if err := internalConfig.Save(cfg); err != nil {
				return errors.New(tr.Tf("error.alias.save_failed", i18n.Args{"message": err.Error()}))
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), tr.Tf("success.alias.deleted", i18n.Args{"name": name}))
			return err
		},
	}
}

func newImportCmd(translators ...*i18n.Translator) *cobra.Command {
	tr := resolve(translators)
	var file string
	cmd := &cobra.Command{
		Use:   "import",
		Short: tr.T("cmd.alias.import.short"),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := readImportSource(cmd.InOrStdin(), file)
			if err != nil {
				return errors.New(tr.Tf("error.alias.import_read", i18n.Args{"message": err.Error()}))
			}
			var pairs map[string]string
			if err := yaml.Unmarshal(data, &pairs); err != nil {
				return errors.New(tr.Tf("error.alias.import_parse", i18n.Args{"message": err.Error()}))
			}

			cfg, err := internalConfig.Load()
			if err != nil {
				return err
			}
			if cfg.Aliases == nil {
				cfg.Aliases = map[string]string{}
			}
			for name, expansion := range pairs {
				cfg.Aliases[name] = expansion
			}
			if err := internalConfig.Save(cfg); err != nil {
				return errors.New(tr.Tf("error.alias.save_failed", i18n.Args{"message": err.Error()}))
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), tr.Tf("success.alias.imported", i18n.Args{"count": len(pairs)}))
			return err
		},
	}
	cmd.Flags().StringVar(&file, "file", "", tr.T("flag.alias.file"))
	return cmd
}

func readImportSource(stdin io.Reader, file string) ([]byte, error) {
	if file != "" {
		return os.ReadFile(file)
	}
	return io.ReadAll(stdin)
}

func resolve(translators []*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
