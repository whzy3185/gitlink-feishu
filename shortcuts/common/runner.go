package common

import (
	"strconv"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/spf13/cobra"
)

// MountShortcut converts a Shortcut into a cobra.Command and adds it as a subcommand.
func MountShortcut(parent *cobra.Command, s *Shortcut, translators ...*i18n.Translator) {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	cmd := &cobra.Command{
		Use:   "+" + s.Name,
		Short: s.Description,
		Long:  s.Long,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Collect flag values
			flagValues := make(map[string]string)
			for _, f := range s.Flags {
				if f.Bool {
					val, _ := cmd.Flags().GetBool(f.Name)
					flagValues[f.Name] = strconv.FormatBool(val)
					continue
				}
				val, _ := cmd.Flags().GetString(f.Name)
				if val != "" {
					flagValues[f.Name] = val
				}
			}

			ctx, err := NewRuntimeContext(flagValues, tr)
			if err != nil {
				return err
			}
			for _, f := range s.Flags {
				if f.Required && flagValues[f.Name] == "" {
					_, err := ctx.RequireArg(f.Name)
					return err
				}
			}

			return s.Run(ctx)
		},
	}

	for _, f := range s.Flags {
		if f.Bool {
			defaultValue, _ := strconv.ParseBool(f.Default)
			if f.Short != "" {
				cmd.Flags().BoolP(f.Name, f.Short, defaultValue, f.Usage)
			} else {
				cmd.Flags().Bool(f.Name, defaultValue, f.Usage)
			}
		} else if f.Short != "" {
			cmd.Flags().StringP(f.Name, f.Short, f.Default, f.Usage)
		} else {
			cmd.Flags().String(f.Name, f.Default, f.Usage)
		}
	}

	parent.AddCommand(cmd)
}

// MountShortcuts mounts multiple shortcuts under a parent command.
func MountShortcuts(parent *cobra.Command, shortcuts []*Shortcut, translators ...*i18n.Translator) {
	var tr *i18n.Translator
	if len(translators) > 0 {
		tr = translators[0]
	}
	for _, s := range shortcuts {
		MountShortcut(parent, s, tr)
	}
}
