package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestCLI_UserManagement(t *testing.T) {
	tmpData := "./test_user_data"
	os.RemoveAll(tmpData)
	defer os.RemoveAll(tmpData)
	os.MkdirAll(tmpData, 0755)

	// 1. Register user
	_, err := executeCommand("register", "--user", "user1", "--pass", "pass1", "--data-dir", tmpData)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// 2. List users
	out, err := executeCommand("user", "list", "--data-dir", tmpData)
	if err != nil {
		t.Fatalf("User list failed: %v", err)
	}
	if !strings.Contains(out, "user1") {
		t.Errorf("Expected user1 in list, got: %s", out)
	}

	// 3. Change password
	_, err = executeCommand("user", "passwd", "--user", "user1", "--pass", "newpass", "--data-dir", tmpData)
	if err != nil {
		t.Fatalf("Passwd failed: %v", err)
	}

	// 4. Delete user (Note: This might be tricky if it asks for confirmation via stdin)
	// For now, let's skip delete or mock stdin if needed.
	// The current implementation uses fmt.Scanln(&confirm)
}

func TestCLI_Filters(t *testing.T) {
	tmpData := "./test_filter_data"
	os.RemoveAll(tmpData)
	defer os.RemoveAll(tmpData)
	os.MkdirAll(tmpData, 0755)

	// Setup user
	executeCommand("register", "--user", "fuser", "--pass", "fpass", "--data-dir", tmpData)

	// 1. Add filter
	out, err := executeCommand("filter", "add", "--user", "fuser", "--pattern", "spam", "--action", "BLOCK", "--data-dir", tmpData)
	if err != nil {
		t.Fatalf("Filter add failed: %v", err)
	}
	if !strings.Contains(out, "Filter 'spam' added") {
		t.Errorf("Unexpected output: %s", out)
	}

	// 2. List filters
	out, err = executeCommand("filter", "list", "--user", "fuser", "--data-dir", tmpData)
	if err != nil {
		t.Fatalf("Filter list failed: %v", err)
	}
	if !strings.Contains(out, "spam") || !strings.Contains(out, "BLOCK") {
		t.Errorf("Filter not found in list: %s", out)
	}

	// Extract ID (it's the first column in the table)
	lines := strings.Split(out, "\n")
	var filterID string
	for _, line := range lines {
		if strings.Contains(line, "spam") {
			parts := strings.Split(line, "|")
			filterID = strings.TrimSpace(parts[0])
			break
		}
	}

	if filterID == "" {
		t.Fatal("Could not find filter ID in list output")
	}

	// 3. Remove filter
	out, err = executeCommand("filter", "remove", "--user", "fuser", "--id", filterID, "--data-dir", tmpData)
	if err != nil {
		t.Fatalf("Filter remove failed: %v", err)
	}
	if !strings.Contains(out, "removed") {
		t.Errorf("Unexpected output: %s", out)
	}
}
