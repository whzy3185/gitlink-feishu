package auth

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	internalAuth "github.com/gitlink-org/gitlink-cli/internal/auth"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
)

const envTokenVar = "GITLINK_TOKEN"

var (
	storeToken = internalAuth.StoreToken
	loadToken  = internalAuth.LoadToken
)

func NewAuthCmd(translators ...*i18n.Translator) *cobra.Command {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	cmd := &cobra.Command{
		Use:   "auth",
		Short: tr.T("cmd.auth.short"),
	}
	cmd.AddCommand(newLoginCmd(tr))
	cmd.AddCommand(newLogoutCmd(tr))
	cmd.AddCommand(newStatusCmd(tr))
	return cmd
}

func newLoginCmd(tr *i18n.Translator) *cobra.Command {
	var tokenMode bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: tr.T("cmd.auth.login.short"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if tokenMode {
				return loginWithToken(cmd.InOrStdin(), cmd.OutOrStdout(), tr)
			}
			return loginWithPassword(cmd.InOrStdin(), cmd.OutOrStdout(), tr)
		},
	}
	cmd.Flags().BoolVar(&tokenMode, "token", false, tr.T("flag.auth.token"))
	return cmd
}

func loginWithPassword(in io.Reader, out io.Writer, tr *i18n.Translator) error {
	reader := bufio.NewReader(in)
	if _, err := fmt.Fprint(out, tr.T("prompt.auth.username")); err != nil {
		return err
	}
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	if _, err := fmt.Fprint(out, tr.T("prompt.auth.password")); err != nil {
		return err
	}
	passwordBytes, err := readPassword(in, reader)
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	password := string(passwordBytes)

	result, err := internalAuth.Login(username, password)
	if err != nil {
		return errors.New(tr.Tf("error.auth.login_failed", i18n.Args{"message": err.Error()}))
	}

	_, err = fmt.Fprintln(out, tr.Tf("success.auth.logged_in_as", i18n.Args{"login": result.Login}))
	return err
}

func readPassword(in io.Reader, reader *bufio.Reader) ([]byte, error) {
	if file, ok := in.(*os.File); ok {
		fd := int(file.Fd())
		if term.IsTerminal(fd) {
			return term.ReadPassword(fd)
		}
	}
	password, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	return []byte(strings.TrimRight(password, "\r\n")), nil
}

func loginWithToken(in io.Reader, out io.Writer, tr *i18n.Translator) error {
	reader := bufio.NewReader(in)
	if _, err := fmt.Fprint(out, tr.T("prompt.auth.token")); err != nil {
		return err
	}
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	if token == "" {
		return errors.New(tr.T("error.auth.token_empty"))
	}

	if err := storeToken(token); err != nil {
		return errors.New(tr.Tf("error.auth.store_token_failed", i18n.Args{"message": err.Error()}))
	}

	_, err := fmt.Fprintln(out, tr.T("success.auth.token_saved"))
	return err
}

func newLogoutCmd(tr *i18n.Translator) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: tr.T("cmd.auth.logout.short"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := internalAuth.DeleteToken(); err != nil {
				return errors.New(tr.Tf("error.auth.delete_token_failed", i18n.Args{"message": err.Error()}))
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), tr.T("success.auth.logged_out"))
			return err
		},
	}
}

func newStatusCmd(tr *i18n.Translator) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: tr.T("cmd.auth.status.short"),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			// Check env var token first
			if envToken := os.Getenv(envTokenVar); envToken != "" {
				if _, err := fmt.Fprintln(out, tr.Tf("success.auth.logged_in_via_env", i18n.Args{"env": envTokenVar})); err != nil {
					return err
				}
			}

			token, err := loadToken()
			if err != nil || token == "" {
				if os.Getenv(envTokenVar) == "" {
					if _, err := fmt.Fprintln(out, tr.T("warning.auth.not_logged_in")); err != nil {
						return err
					}
					if _, err := fmt.Fprintln(out, tr.T("output.auth.login_hint")); err != nil {
						return err
					}
					if _, err := fmt.Fprintln(out, tr.Tf("output.auth.env_hint", i18n.Args{"env": envTokenVar})); err != nil {
						return err
					}
				}
				return nil
			}

			user, err := internalAuth.GetCurrentUser()
			if err != nil {
				_, err := fmt.Fprintln(out, tr.Tf("warning.auth.token_unverified", i18n.Args{"message": err.Error()}))
				return err
			}

			login, _ := user["login"].(string)
			name, _ := user["name"].(string)
			if login != "" {
				text := tr.Tf("success.auth.logged_in_as", i18n.Args{"login": login})
				if name != "" {
					text = fmt.Sprintf("%s (%s)", text, name)
				}
				_, err := fmt.Fprintln(out, text)
				return err
			}
			_, err = fmt.Fprintln(out, tr.T("warning.auth.user_unavailable"))
			return err
		},
	}
}
