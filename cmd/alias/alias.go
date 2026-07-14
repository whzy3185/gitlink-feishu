package alias

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/internal/config"
	"github.com/gitlink-org/gitlink-cli/internal/output"
)

// AliasConfig represents the aliases section of the CLI config.
type AliasConfig struct {
	Aliases map[string]string `yaml:"aliases,omitempty"`
}

// NewAliasCmd creates the alias command with subcommands.
func NewAliasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alias",
		Short: "管理命令别名（把长命令变短）",
		Long: `管理 gitlink-cli 的命令别名。

别名允许你为常用命令创建简短的名称，例如：
  gitlink-cli alias +set rl "repo +list"
  之后可以使用: gitlink-cli rl

别名存储在 ~/.config/gitlink-cli/aliases.yaml 中。`,
		Example: `  gitlink-cli alias +list
  gitlink-cli alias +set rl "repo +list"
  gitlink-cli alias +set ri "repo +info --owner Gitlink --repo gitlink-cli"
  gitlink-cli alias +delete rl`,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "+list",
			Short: "列出所有已定义的别名",
			Long:  "列出所有已定义的命令别名。如果没有任何别名，会给出创建提示。",
			RunE: func(cmd *cobra.Command, args []string) error {
				aliases, _ := loadAliases()
				// Structured formats (json/yaml/table) route through output.Print
				// so alias +list integrates with scripts and AI Agents.
				if cmdutil.Format == "json" || cmdutil.Format == "yaml" || cmdutil.Format == "table" {
					rows := make([]map[string]string, 0, len(aliases))
					names := make([]string, 0, len(aliases))
					for k := range aliases {
						names = append(names, k)
					}
					sort.Strings(names)
					for _, k := range names {
						rows = append(rows, map[string]string{"name": k, "command": aliases[k]})
					}
					return output.Print(output.SuccessEnvelope(rows, nil), cmdutil.Format)
				}
				if len(aliases) == 0 {
					fmt.Println("（未定义任何别名）")
					fmt.Println("使用 alias +set <名称> <命令> 来创建别名")
					return nil
				}
				names := make([]string, 0, len(aliases))
				for k := range aliases {
					names = append(names, k)
				}
				sort.Strings(names)
				for _, k := range names {
					fmt.Printf("  %-15s → %s\n", k, aliases[k])
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "+set <name> <command>",
			Short: "设置别名",
			Long:  "为一条命令设置别名。如果别名已存在，会被覆盖。",
			Args:  cobra.ExactArgs(2),
			Example: `  gitlink-cli alias +set rl "repo +list"
  gitlink-cli alias +set ri "repo +info"`,
			RunE: func(cmd *cobra.Command, args []string) error {
				aliases, _ := loadAliases()
				aliases[args[0]] = args[1]
				if err := saveAliases(aliases); err != nil {
					return err
				}
				fmt.Printf("别名已设置: %s → %s\n", args[0], args[1])
				return nil
			},
		},
		&cobra.Command{
			Use:   "+delete <name>",
			Short: "删除别名",
			Long:  "删除一个已定义的命令别名。",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				aliases, _ := loadAliases()
				if _, ok := aliases[args[0]]; !ok {
					return fmt.Errorf("别名 %s 不存在", args[0])
				}
				delete(aliases, args[0])
				if err := saveAliases(aliases); err != nil {
					return err
				}
				fmt.Printf("别名已删除: %s\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use:   "+expand <name>",
			Short: "展开别名查看原命令",
			Long:  "查看一个别名对应的原始命令。如果别名不存在则报错。",
			Args:  cobra.ExactArgs(1),
			Example: `  gitlink-cli alias +expand rl
	  输出: rl → repo +list`,
			RunE: func(cmd *cobra.Command, args []string) error {
				aliases, _ := loadAliases()
				expanded, ok := aliases[args[0]]
				if !ok {
					return fmt.Errorf("别名 %s 不存在", args[0])
				}
				fmt.Printf("%s → %s\n", args[0], expanded)
				return nil
			},
		},
	)
	return cmd
}

func aliasesPath() string {
	return config.ConfigDir() + "/aliases.yaml"
}

func loadAliases() (map[string]string, error) {
	data, err := os.ReadFile(aliasesPath())
	if err != nil {
		return make(map[string]string), nil
	}
	var ac AliasConfig
	if err := yaml.Unmarshal(data, &ac); err != nil {
		return make(map[string]string), nil
	}
	if ac.Aliases == nil {
		ac.Aliases = make(map[string]string)
	}
	return ac.Aliases, nil
}

func saveAliases(a map[string]string) error {
	data, err := yaml.Marshal(AliasConfig{Aliases: a})
	if err != nil {
		return err
	}
	os.MkdirAll(config.ConfigDir(), 0700)
	return os.WriteFile(aliasesPath(), data, 0600)
}
