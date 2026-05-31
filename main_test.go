package main

import (
	"os"
	"testing"
)

func TestMainHelp(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"gitlink-cli", "--help"}
	defer func() { os.Args = origArgs }()

	// Should not call os.Exit because --help returns nil
	main()
}
