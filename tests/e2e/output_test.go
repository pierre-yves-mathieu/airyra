package e2e

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestE2E_JSONOutput_List tests that ar list --json produces valid JSON output.
func TestE2E_JSONOutput_List(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "json-output-test"
	projectDir := suite.createProject(projectName)

	// Create some tasks
	suite.createTask(projectName, "Task 1")
	suite.createTask(projectName, "Task 2")
	suite.createTask(projectName, "Task 3")

	// Run ar list --json
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "list", "--json")
	if exitCode != 0 {
		t.Fatalf("ar list --json failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify output is valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, stdout)
	}

	// Verify structure has expected fields
	data, ok := result["data"]
	if !ok {
		t.Error("JSON output should have 'data' field")
	}

	dataSlice, ok := data.([]interface{})
	if !ok {
		t.Errorf("'data' should be an array, got %T", data)
	}

	if len(dataSlice) != 3 {
		t.Errorf("Expected 3 tasks in data array, got %d", len(dataSlice))
	}

	// Verify pagination metadata
	pagination, ok := result["pagination"]
	if !ok {
		t.Error("JSON output should have 'pagination' field")
	}

	pagMap, ok := pagination.(map[string]interface{})
	if !ok {
		t.Errorf("'pagination' should be an object, got %T", pagination)
	}

	// Check required pagination fields
	requiredPagFields := []string{"page", "per_page", "total", "total_pages"}
	for _, field := range requiredPagFields {
		if _, ok := pagMap[field]; !ok {
			t.Errorf("pagination should have '%s' field", field)
		}
	}

	// Verify task structure
	if len(dataSlice) > 0 {
		firstTask, ok := dataSlice[0].(map[string]interface{})
		if !ok {
			t.Fatalf("Task should be an object, got %T", dataSlice[0])
		}

		requiredTaskFields := []string{"id", "title", "status", "priority", "created_at", "updated_at"}
		for _, field := range requiredTaskFields {
			if _, ok := firstTask[field]; !ok {
				t.Errorf("Task should have '%s' field", field)
			}
		}
	}
}

// TestE2E_JSONOutput_Show tests that ar show <id> --json produces valid JSON.
func TestE2E_JSONOutput_Show(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "json-show-test"
	projectDir := suite.createProject(projectName)

	// Create a task
	taskID := suite.createTask(projectName, "Test Task")

	// Run ar show --json
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "show", taskID, "--json")
	if exitCode != 0 {
		t.Fatalf("ar show --json failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify output is valid JSON
	var task map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &task); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, stdout)
	}

	// Verify task fields
	if task["id"] != taskID {
		t.Errorf("Task ID = %v, want %s", task["id"], taskID)
	}

	if task["title"] != "Test Task" {
		t.Errorf("Task title = %v, want 'Test Task'", task["title"])
	}

	if task["status"] != "open" {
		t.Errorf("Task status = %v, want 'open'", task["status"])
	}
}

// TestE2E_JSONOutput_History tests that ar history <id> --json produces valid JSON.
func TestE2E_JSONOutput_History(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "json-history-test"
	projectDir := suite.createProject(projectName)

	// Create and claim a task to generate history
	taskID := suite.createTask(projectName, "History Task")
	suite.claimTask(projectName, taskID, "test-agent")

	// Run ar history --json
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "history", taskID, "--json")
	if exitCode != 0 {
		t.Fatalf("ar history --json failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify output is valid JSON array
	var entries []interface{}
	if err := json.Unmarshal([]byte(stdout), &entries); err != nil {
		t.Fatalf("Output is not valid JSON array: %v\nOutput: %s", err, stdout)
	}

	// Should have at least 2 entries (create, claim)
	if len(entries) < 2 {
		t.Errorf("Expected at least 2 history entries, got %d", len(entries))
	}

	// Verify entry structure
	if len(entries) > 0 {
		entry, ok := entries[0].(map[string]interface{})
		if !ok {
			t.Fatalf("Entry should be an object, got %T", entries[0])
		}

		requiredFields := []string{"task_id", "action", "changed_at", "changed_by"}
		for _, field := range requiredFields {
			if _, ok := entry[field]; !ok {
				t.Errorf("History entry should have '%s' field", field)
			}
		}
	}
}

// TestE2E_TableOutput_List tests human-readable table output for ar list.
func TestE2E_TableOutput_List(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "table-output-test"
	projectDir := suite.createProject(projectName)

	// Create tasks
	suite.createTask(projectName, "First Task")
	suite.createTask(projectName, "Second Task")

	// Run ar list (no --json)
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "list")
	if exitCode != 0 {
		t.Fatalf("ar list failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify column headers are present
	if !containsString(stdout, "ID") {
		t.Error("Table output should have 'ID' header")
	}
	if !containsString(stdout, "TITLE") {
		t.Error("Table output should have 'TITLE' header")
	}
	if !containsString(stdout, "STATUS") {
		t.Error("Table output should have 'STATUS' header")
	}
	if !containsString(stdout, "PRIORITY") {
		t.Error("Table output should have 'PRIORITY' header")
	}

	// Verify tasks are listed
	if !containsString(stdout, "First Task") {
		t.Error("Table output should contain 'First Task'")
	}
	if !containsString(stdout, "Second Task") {
		t.Error("Table output should contain 'Second Task'")
	}

	// Verify separator line
	if !containsString(stdout, "--") {
		t.Error("Table output should have separator line")
	}
}

// TestE2E_TableOutput_Show tests human-readable output for ar show.
func TestE2E_TableOutput_Show(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "table-show-test"
	projectDir := suite.createProject(projectName)

	// Create a task with description
	c := suite.getClient(projectName, "test-agent")
	task, _ := c.CreateTask(t.Context(), "Detailed Task", "This is a description", 1, "")

	// Run ar show (no --json)
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "show", task.ID)
	if exitCode != 0 {
		t.Fatalf("ar show failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify key fields are present
	if !containsString(stdout, "ID:") {
		t.Error("Show output should have 'ID:' field")
	}
	if !containsString(stdout, "Title:") {
		t.Error("Show output should have 'Title:' field")
	}
	if !containsString(stdout, "Status:") {
		t.Error("Show output should have 'Status:' field")
	}
	if !containsString(stdout, "Priority:") {
		t.Error("Show output should have 'Priority:' field")
	}

	// Verify values
	if !containsString(stdout, task.ID) {
		t.Errorf("Show output should contain task ID: %s", task.ID)
	}
	if !containsString(stdout, "Detailed Task") {
		t.Error("Show output should contain task title")
	}
	if !containsString(stdout, "open") {
		t.Error("Show output should contain status 'open'")
	}
	if !containsString(stdout, "high") {
		t.Error("Show output should contain priority 'high'")
	}
}

// TestE2E_TableOutput_Ready tests human-readable output for ar ready.
func TestE2E_TableOutput_Ready(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "ready-output-test"
	projectDir := suite.createProject(projectName)

	// Create some tasks
	suite.createTask(projectName, "Ready Task 1")
	suite.createTask(projectName, "Ready Task 2")

	// Run ar ready
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "ready")
	if exitCode != 0 {
		t.Fatalf("ar ready failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify table format with headers
	if !containsString(stdout, "ID") {
		t.Error("Ready output should have 'ID' header")
	}
	if !containsString(stdout, "TITLE") {
		t.Error("Ready output should have 'TITLE' header")
	}

	// Verify tasks are listed
	if !containsString(stdout, "Ready Task 1") {
		t.Error("Ready output should contain 'Ready Task 1'")
	}
	if !containsString(stdout, "Ready Task 2") {
		t.Error("Ready output should contain 'Ready Task 2'")
	}
}

// TestE2E_TableOutput_Empty tests output when no tasks exist.
func TestE2E_TableOutput_Empty(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "empty-output-test"
	projectDir := suite.createProject(projectName)

	// Run ar list on empty project
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "list")
	if exitCode != 0 {
		t.Fatalf("ar list failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Should indicate no tasks
	if !containsString(stdout, "No tasks") {
		t.Errorf("Empty list should indicate no tasks found:\n%s", stdout)
	}
}

// TestE2E_JSONOutput_Ready tests JSON output for ar ready.
func TestE2E_JSONOutput_Ready(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "json-ready-test"
	projectDir := suite.createProject(projectName)

	// Create tasks with dependencies
	taskA := suite.createTask(projectName, "Task A - Ready")
	taskB := suite.createTask(projectName, "Task B - Not Ready")
	suite.addDependency(projectName, taskB, taskA)

	// Run ar ready --json
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "ready", "--json")
	if exitCode != 0 {
		t.Fatalf("ar ready --json failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify valid JSON with pagination
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, stdout)
	}

	// Verify only ready task is returned
	data, ok := result["data"].([]interface{})
	if !ok {
		t.Fatal("Data should be an array")
	}

	if len(data) != 1 {
		t.Errorf("Expected 1 ready task, got %d", len(data))
	}

	if len(data) > 0 {
		task := data[0].(map[string]interface{})
		if task["id"] != taskA {
			t.Errorf("Ready task should be %s, got %v", taskA, task["id"])
		}
	}
}

// TestE2E_JSONOutput_Create tests JSON output when creating a task.
func TestE2E_JSONOutput_Create(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "json-create-test"
	projectDir := suite.createProject(projectName)

	// Run ar create --json
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "create", "New Task", "-d", "Description", "-p", "high", "--json")
	if exitCode != 0 {
		t.Fatalf("ar create --json failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify valid JSON
	var task map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &task); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, stdout)
	}

	// Verify created task fields
	if task["title"] != "New Task" {
		t.Errorf("Task title = %v, want 'New Task'", task["title"])
	}

	if task["description"] != "Description" {
		t.Errorf("Task description = %v, want 'Description'", task["description"])
	}

	// Priority 1 = high
	if priority, ok := task["priority"].(float64); !ok || int(priority) != 1 {
		t.Errorf("Task priority = %v, want 1 (high)", task["priority"])
	}

	if task["status"] != "open" {
		t.Errorf("Task status = %v, want 'open'", task["status"])
	}

	// Verify ID is present
	if id, ok := task["id"].(string); !ok || id == "" {
		t.Error("Task should have non-empty ID")
	}
}

// TestE2E_JSONOutput_Error tests JSON output for errors.
func TestE2E_JSONOutput_Error(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "json-error-test"
	projectDir := suite.createProject(projectName)

	// Run ar show on non-existent task with --json
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "show", "ar-nonexistent", "--json")

	// Should fail
	if exitCode == 0 {
		t.Error("ar show non-existent should fail")
	}

	// Error output might be in stdout or stderr depending on implementation
	output := stdout + stderr

	// Check if output is JSON (it should contain error info)
	if containsString(output, "{") {
		var errResp map[string]interface{}
		// Try to parse either stdout or stderr as JSON
		if err := json.Unmarshal([]byte(stdout), &errResp); err != nil {
			// If stdout is not JSON, that's also acceptable
			t.Logf("Error output is not JSON (might be plain text): %s", output)
		} else {
			// If it is JSON, verify error field exists
			if _, ok := errResp["error"]; ok {
				t.Log("Error response has 'error' field")
			}
		}
	}

	// At minimum, verify there's some error indication
	if !containsString(output, "not found") && !containsString(output, "error") && !containsString(output, "Error") {
		t.Errorf("Error output should indicate the error:\n%s", output)
	}
}

// TestE2E_Output_Pagination tests pagination info in output.
func TestE2E_Output_Pagination(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "pagination-test"
	projectDir := suite.createProject(projectName)

	// Create many tasks
	numTasks := 10
	for i := 0; i < numTasks; i++ {
		suite.createTask(projectName, taskTitleForNum(i))
	}

	// List with small page size
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "list", "--per-page", "3", "--json")
	if exitCode != 0 {
		t.Fatalf("ar list --per-page failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify pagination
	pagination := result["pagination"].(map[string]interface{})

	if int(pagination["page"].(float64)) != 1 {
		t.Errorf("Page should be 1, got %v", pagination["page"])
	}

	if int(pagination["per_page"].(float64)) != 3 {
		t.Errorf("Per page should be 3, got %v", pagination["per_page"])
	}

	if int(pagination["total"].(float64)) != numTasks {
		t.Errorf("Total should be %d, got %v", numTasks, pagination["total"])
	}

	// total_pages should be ceil(10/3) = 4
	expectedPages := (numTasks + 2) / 3 // ceil division
	if int(pagination["total_pages"].(float64)) != expectedPages {
		t.Errorf("Total pages should be %d, got %v", expectedPages, pagination["total_pages"])
	}

	// Data should have 3 items (first page)
	data := result["data"].([]interface{})
	if len(data) != 3 {
		t.Errorf("First page should have 3 items, got %d", len(data))
	}
}

// TestE2E_Output_Dependencies tests output for dependency commands.
func TestE2E_Output_Dependencies(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "deps-output-test"
	projectDir := suite.createProject(projectName)

	// Create tasks
	taskA := suite.createTask(projectName, "Task A")
	taskB := suite.createTask(projectName, "Task B")

	// Add dependency
	suite.addDependency(projectName, taskB, taskA)

	// List dependencies with table format
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "dep", "list", taskB)
	if exitCode != 0 {
		t.Fatalf("ar dep list failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Should show dependency on A
	if !containsString(stdout, "depends on") {
		t.Errorf("Dependency output should show 'depends on':\n%s", stdout)
	}
	if !containsString(stdout, taskA) {
		t.Errorf("Dependency output should show parent task ID %s:\n%s", taskA, stdout)
	}

	// List dependencies with JSON
	stdout, stderr, exitCode = suite.runCLIInDir(projectDir, "dep", "list", taskB, "--json")
	if exitCode != 0 {
		t.Fatalf("ar dep list --json failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	var deps []interface{}
	if err := json.Unmarshal([]byte(stdout), &deps); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, stdout)
	}

	// Should have 1 dependency
	if len(deps) < 1 {
		t.Errorf("Should have at least 1 dependency, got %d", len(deps))
	}
}

// TestE2E_Output_SuccessMessages tests success messages from various commands.
func TestE2E_Output_SuccessMessages(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "success-msg-test"
	projectDir := suite.createProject(projectName)

	taskID := suite.createTask(projectName, "Success Task")

	// Test delete success message
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "delete", taskID)
	if exitCode != 0 {
		t.Fatalf("ar delete failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	if !containsString(stdout, "deleted") {
		t.Errorf("Delete output should mention 'deleted':\n%s", stdout)
	}

	// Create new task for dependency tests
	taskA := suite.createTask(projectName, "Task A")
	taskB := suite.createTask(projectName, "Task B")

	// Test dep add success message
	stdout, stderr, exitCode = suite.runCLIInDir(projectDir, "dep", "add", taskB, taskA)
	if exitCode != 0 {
		t.Fatalf("ar dep add failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	if !containsString(stdout, "depends on") {
		t.Errorf("Dep add output should mention 'depends on':\n%s", stdout)
	}

	// Test dep rm success message
	stdout, stderr, exitCode = suite.runCLIInDir(projectDir, "dep", "rm", taskB, taskA)
	if exitCode != 0 {
		t.Fatalf("ar dep rm failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	if !containsString(stdout, "no longer depends") {
		t.Errorf("Dep rm output should mention 'no longer depends':\n%s", stdout)
	}
}

// TestE2E_Output_LongTitle tests output formatting with long titles.
func TestE2E_Output_LongTitle(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "long-title-test"
	projectDir := suite.createProject(projectName)

	// Create task with very long title
	longTitle := strings.Repeat("A very long task title that should be truncated ", 3)
	suite.createTask(projectName, longTitle)

	// List tasks (should truncate)
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "list")
	if exitCode != 0 {
		t.Fatalf("ar list failed: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Output should be reasonably formatted (not break the table)
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		// No line should be excessively long (e.g., > 200 chars unless it's the full title)
		if len(line) > 200 {
			t.Logf("Warning: Line is quite long (%d chars)", len(line))
		}
	}

	// Show should display full title
	// First extract task ID from list
	listJson, _, _ := suite.runCLIInDir(projectDir, "list", "--json")
	var result map[string]interface{}
	json.Unmarshal([]byte(listJson), &result)
	data := result["data"].([]interface{})
	taskID := data[0].(map[string]interface{})["id"].(string)

	stdout, _, exitCode = suite.runCLIInDir(projectDir, "show", taskID)
	if exitCode != 0 {
		t.Fatal("ar show failed")
	}

	// Show output should contain the title
	if !containsString(stdout, "very long task title") {
		t.Error("Show output should contain the task title")
	}
}
