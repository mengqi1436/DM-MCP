package tools

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBuildCSVControlFile(t *testing.T) {
	file := csvImportFile{
		CSVFile: "/data/users.csv",
		Schema:  "APP",
		Table:   "USERS",
		Columns: []string{"ID", "NAME"},
		Skip:    1,
	}
	options := csvImportOptions{
		Delimiter:     ",",
		EnclosedBy:    "\"",
		CharacterCode: "UTF-8",
		Rows:          50000,
		Direct:        true,
		IndexOption:   2,
		Mode:          "APPEND",
	}

	control := buildCSVControlFile(options, file, "/data/users.bad")

	assertContains(t, control, "OPTIONS")
	assertContains(t, control, "SKIP = 1")
	assertContains(t, control, "ROWS = 50000")
	assertContains(t, control, "DIRECT = TRUE")
	assertContains(t, control, "INDEX_OPTION = 2")
	assertContains(t, control, "CHARACTER_CODE='UTF-8'")
	assertContains(t, control, "INFILE '/data/users.csv'")
	assertContains(t, control, "BADFILE '/data/users.bad'")
	assertContains(t, control, "APPEND\nINTO TABLE APP.USERS")
	assertContains(t, control, "FIELDS TERMINATED BY ','")
	assertContains(t, control, "OPTIONALLY ENCLOSED BY '\"'")
	assertContains(t, control, "ID")
	assertContains(t, control, "NAME")
}

func TestParseCSVImportOptionsValidatesInputs(t *testing.T) {
	tempDir := t.TempDir()
	csvPath := filepath.Join(tempDir, "users.csv")
	if err := os.WriteFile(csvPath, []byte("ID,NAME\n1,Alice\n"), 0600); err != nil {
		t.Fatalf("write csv fixture: %v", err)
	}

	params := map[string]interface{}{
		"work_dir": tempDir,
		"files": []interface{}{
			map[string]interface{}{
				"csv_file": csvPath,
				"table":    "APP.USERS",
				"columns":  []interface{}{"ID", "NAME"},
				"skip":     float64(1),
			},
		},
	}

	options, err := parseCSVImportOptions(params)
	if err != nil {
		t.Fatalf("parse options failed: %v", err)
	}

	if options.WorkDir != tempDir {
		t.Fatalf("work dir = %q, want %q", options.WorkDir, tempDir)
	}
	if options.MaxParallel != 1 {
		t.Fatalf("max parallel = %d, want 1", options.MaxParallel)
	}
	if options.Files[0].Schema != "APP" || options.Files[0].Table != "USERS" {
		t.Fatalf("parsed table = %s.%s, want APP.USERS", options.Files[0].Schema, options.Files[0].Table)
	}
	if len(options.Files[0].Columns) != 2 {
		t.Fatalf("columns length = %d, want 2", len(options.Files[0].Columns))
	}
}

func TestParseCSVImportOptionsRejectsUnsafeIdentifier(t *testing.T) {
	tempDir := t.TempDir()
	csvPath := filepath.Join(tempDir, "users.csv")
	if err := os.WriteFile(csvPath, []byte("ID\n1\n"), 0600); err != nil {
		t.Fatalf("write csv fixture: %v", err)
	}

	_, err := parseCSVImportOptions(map[string]interface{}{
		"work_dir": tempDir,
		"files": []interface{}{
			map[string]interface{}{
				"csv_file": csvPath,
				"table":    "USERS;DROP",
			},
		},
	})
	if err == nil {
		t.Fatal("expected unsafe identifier error")
	}
}

func TestRunCSVImportsLimitsParallelismAndRedactsOutput(t *testing.T) {
	t.Setenv("DM_PASSWORD", "secret")
	tempDir := t.TempDir()
	files := make([]csvImportFile, 3)
	for i := range files {
		csvPath := filepath.Join(tempDir, "users_"+strconv.Itoa(i)+".csv")
		if err := os.WriteFile(csvPath, []byte("ID\n1\n"), 0600); err != nil {
			t.Fatalf("write csv fixture: %v", err)
		}
		files[i] = csvImportFile{
			CSVFile: csvPath,
			Table:   "USERS",
		}
	}

	options := csvImportOptions{
		Files:         files,
		DMFLDRPath:    "dmfldr",
		WorkDir:       tempDir,
		Delimiter:     ",",
		CharacterCode: "UTF-8",
		Rows:          50000,
		Direct:        true,
		IndexOption:   2,
		Mode:          "APPEND",
		MaxParallel:   2,
	}

	var current int32
	var maxSeen int32
	runner := func(ctx context.Context, dmfldrPath string, args []string) (string, error) {
		now := atomic.AddInt32(&current, 1)
		for {
			previous := atomic.LoadInt32(&maxSeen)
			if now <= previous || atomic.CompareAndSwapInt32(&maxSeen, previous, now) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt32(&current, -1)
		return "loaded with password secret", nil
	}

	results := runCSVImports(options, runner)

	if atomic.LoadInt32(&maxSeen) > 2 {
		t.Fatalf("max parallelism = %d, want <= 2", maxSeen)
	}
	for _, result := range results {
		if result.Status != "success" {
			t.Fatalf("status = %q, want success: %#v", result.Status, result)
		}
		if strings.Contains(result.Output, "secret") {
			t.Fatalf("output was not redacted: %q", result.Output)
		}
		if result.ControlFile == "" || result.LogFile == "" || result.BadFile == "" {
			t.Fatalf("missing output paths: %#v", result)
		}
	}
}

func assertContains(t *testing.T, value, want string) {
	t.Helper()
	if !strings.Contains(value, want) {
		t.Fatalf("expected %q to contain %q", value, want)
	}
}
