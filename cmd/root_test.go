package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func executeCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	// In Cobra, we need to reset flags to avoid pollution between tests
	// as PersistentFlags and sub-command flags are persistent in the global rootCmd.
	for _, cmd := range rootCmd.Commands() {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			f.Value.Set(f.DefValue)
		})
	}
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
	})

	err := rootCmd.Execute()
	return buf.String(), err
}

func TestCLI_Flow(t *testing.T) {
	// Setup a temporary data directory
	tmpData := "./test_cli_data"
	os.RemoveAll(tmpData)
	defer os.RemoveAll(tmpData)
	os.MkdirAll(tmpData, 0755)

	// 1. Test Register
	out, err := executeCommand("register", "--user", "tester", "--pass", "password", "--data-dir", tmpData)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if !strings.Contains(out, "registered") {
		t.Errorf("Expected registration message, got: %s", out)
	}

	// 2. Test List (Empty)
	out, err = executeCommand("list", "--user", "tester", "--data-dir", tmpData)
	if err != nil {
		t.Errorf("List failed: %v", err)
	}
	if !strings.Contains(out, "Found: 0") {
		t.Errorf("Expected 0 records, got: %s", out)
	}

	// 3. Test Add
	out, err = executeCommand("add", "--url", "https://news.ycombinator.com/rss", "--user", "tester", "--data-dir", tmpData)
	if err != nil {
		t.Logf("Add might fail due to network: %v", err)
	}
}
