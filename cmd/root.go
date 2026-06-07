package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	apiCmd "github.com/gitlink-org/gitlink-cli/cmd/api"
	authCmd "github.com/gitlink-org/gitlink-cli/cmd/auth"
	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	configCmd "github.com/gitlink-org/gitlink-cli/cmd/config"
	doctorCmd "github.com/gitlink-org/gitlink-cli/cmd/doctor"
	internalConfig "github.com/gitlink-org/gitlink-cli/internal/config"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts"
)

var Version = "dev"

type RootOptions struct {
	Version    string
	Args       []string
	Env        map[string]string
	ConfigLang string
}

func NewRootCmd(opts RootOptions, tr *i18n.Translator) (*cobra.Command, error) {
	if tr == nil {
		var err error
		tr, err = newTranslator(opts.Args, opts.Env, opts.ConfigLang)
		if err != nil {
			return nil, err
		}
	}

	version := opts.Version
	if version == "" {
		version = Version
	}

	rootCmd := &cobra.Command{
		Use:           "gitlink-cli",
		Short:         tr.T("cmd.root.short"),
		Long:          tr.T("cmd.root.long"),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&cmdutil.Owner, "owner", "", tr.T("flag.owner"))
	rootCmd.PersistentFlags().StringVar(&cmdutil.Repo, "repo", "", tr.T("flag.repo"))
	rootCmd.PersistentFlags().StringVar(&cmdutil.Format, "format", "", tr.T("flag.format"))
	rootCmd.PersistentFlags().BoolVar(&cmdutil.Debug, "debug", false, tr.T("flag.debug"))
	rootCmd.PersistentFlags().StringVar(&cmdutil.Lang, "lang", "", tr.T("flag.lang"))

	rootCmd.AddCommand(authCmd.NewAuthCmd(tr))
	rootCmd.AddCommand(apiCmd.NewAPICmd(tr))
	rootCmd.AddCommand(configCmd.NewConfigCmd(tr))
	rootCmd.AddCommand(doctorCmd.NewDoctorCmd(tr))
	rootCmd.AddCommand(newVersionCmd(version, tr))

	shortcuts.RegisterAll(rootCmd, tr)

	if opts.Args != nil {
		rootCmd.SetArgs(opts.Args)
	}
	return rootCmd, nil
}

func newVersionCmd(version string, tr *i18n.Translator) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: tr.T("cmd.version.short"),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), tr.Tf("output.version", i18n.Args{"version": version}))
			return err
		},
	}
}

func Execute() error {
	args := os.Args[1:]
	rootCmd, err := NewRootCmd(RootOptions{
		Version: Version,
		Args:    args,
	}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

func newTranslator(args []string, env map[string]string, configLang string) (*i18n.Translator, error) {
	available, err := i18n.AvailableLocales()
	if err != nil {
		return nil, err
	}
	if env == nil {
		env = i18n.EnvMap()
	}
	if configLang == "" {
		configLang = loadConfigLangBestEffort()
	}
	resolved := i18n.ResolveLocaleDetailed(i18n.ResolveOptions{
		ExplicitLang: i18n.PreScanLang(args),
		Env:          env,
		ConfigLang:   configLang,
	}, available)
	if !resolved.Supported && (resolved.Source == "flag" || resolved.Source == "env") {
		tr := i18n.Default()
		return nil, errors.New(tr.Tf("error.unsupported_language", i18n.Args{"lang": resolved.Requested}))
	}
	return i18n.New(i18n.Options{Locale: resolved.Locale})
}

func loadConfigLangBestEffort() string {
	cfg, err := internalConfig.Load()
	if err != nil {
		return ""
	}
	return cfg.Lang
}
