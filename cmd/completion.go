package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	var noDescriptions bool
	command := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion scripts",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(cmd.OutOrStdout(), !noDescriptions)
			case "zsh":
				if noDescriptions {
					return root.GenZshCompletionNoDesc(cmd.OutOrStdout())
				}
				return root.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return root.GenFishCompletion(cmd.OutOrStdout(), !noDescriptions)
			case "powershell":
				if noDescriptions {
					return root.GenPowerShellCompletion(cmd.OutOrStdout())
				}
				return root.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("invalid argument %q for completion", args[0])
			}
		},
	}
	command.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "Disable completion descriptions")
	return command
}
