package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func init() {
	registerBackupTools()
}

func registerBackupTools() {
	RegisterTool(ToolInfo{
		Name:        "logical_export",
		Category:    "backup",
		Description: "逻辑导出（dexp）。参数: output_file(必填), directory(可选), owner(可选)-按用户导出, tables(可选)-表名数组, full(可选)-全库导出, dexp_path(可选), timeout_seconds(可选)",
		Params:      []string{"output_file", "directory", "owner", "tables", "full", "dexp_path", "timeout_seconds"},
	}, handleLogicalExport)

	RegisterTool(ToolInfo{
		Name:        "logical_import",
		Category:    "backup",
		Description: "逻辑导入（dimp）。参数: input_file(必填), directory(可选), owner(可选), tables(可选)-表名数组, full(可选), dimp_path(可选), timeout_seconds(可选)",
		Params:      []string{"input_file", "directory", "owner", "tables", "full", "dimp_path", "timeout_seconds"},
	}, handleLogicalImport)

	RegisterTool(ToolInfo{
		Name:        "physical_backup",
		Category:    "backup",
		Description: "物理备份（dmrman）。参数: backup_dir(必填), dm_ini_path(必填)-dm.ini路径, backup_name(可选), dmrman_path(可选), timeout_seconds(可选)",
		Params:      []string{"backup_dir", "dm_ini_path", "backup_name", "dmrman_path", "timeout_seconds"},
	}, handlePhysicalBackup)

	RegisterTool(ToolInfo{
		Name:        "physical_restore",
		Category:    "backup",
		Description: "物理恢复（dmrman）。参数: backup_dir(必填), dm_ini_path(必填)-dm.ini路径, dmrman_path(可选), timeout_seconds(可选), confirm(必填)-必须为true",
		Params:      []string{"backup_dir", "dm_ini_path", "dmrman_path", "timeout_seconds", "confirm"},
	}, handlePhysicalRestore)
}

func handleLogicalExport(params map[string]interface{}) (interface{}, error) {
	outputFile := getString(params, "output_file")
	if outputFile == "" {
		return nil, fmt.Errorf("参数 output_file 是必需的")
	}

	dexpPath := firstNonEmpty(getString(params, "dexp_path"), os.Getenv("DM_EXP_PATH"), "dexp")
	timeoutSeconds := getInt(params, "timeout_seconds", 300)

	cfg := getConfigForBackup()
	userID := fmt.Sprintf("%s/%s@%s:%d", cfg.user, cfg.password, cfg.host, cfg.port)

	args := []string{userID, "FILE=" + outputFile}

	if dir := getString(params, "directory"); dir != "" {
		args = append(args, "DIRECTORY="+dir)
	}
	if owner := getString(params, "owner"); owner != "" {
		args = append(args, "OWNER="+owner)
	}
	if tables, err := parseStringArray(params, "tables"); err == nil && len(tables) > 0 {
		args = append(args, "TABLES="+strings.Join(tables, ","))
	}
	if getBool(params, "full") {
		args = append(args, "FULL=Y")
	}

	output, err := runBackupCommand(dexpPath, args, timeoutSeconds, cfg.password)
	if err != nil {
		return nil, fmt.Errorf("逻辑导出失败: %w; output=%s", err, output)
	}

	return map[string]interface{}{
		"success":     true,
		"output_file": outputFile,
		"output":      output,
	}, nil
}

func handleLogicalImport(params map[string]interface{}) (interface{}, error) {
	inputFile := getString(params, "input_file")
	if inputFile == "" {
		return nil, fmt.Errorf("参数 input_file 是必需的")
	}

	dimpPath := firstNonEmpty(getString(params, "dimp_path"), os.Getenv("DM_IMP_PATH"), "dimp")
	timeoutSeconds := getInt(params, "timeout_seconds", 300)

	cfg := getConfigForBackup()
	userID := fmt.Sprintf("%s/%s@%s:%d", cfg.user, cfg.password, cfg.host, cfg.port)

	args := []string{userID, "FILE=" + inputFile}

	if dir := getString(params, "directory"); dir != "" {
		args = append(args, "DIRECTORY="+dir)
	}
	if owner := getString(params, "owner"); owner != "" {
		args = append(args, "OWNER="+owner)
	}
	if tables, err := parseStringArray(params, "tables"); err == nil && len(tables) > 0 {
		args = append(args, "TABLES="+strings.Join(tables, ","))
	}
	if getBool(params, "full") {
		args = append(args, "FULL=Y")
	}

	output, err := runBackupCommand(dimpPath, args, timeoutSeconds, cfg.password)
	if err != nil {
		return nil, fmt.Errorf("逻辑导入失败: %w; output=%s", err, output)
	}

	return map[string]interface{}{
		"success":    true,
		"input_file": inputFile,
		"output":     output,
	}, nil
}

func handlePhysicalBackup(params map[string]interface{}) (interface{}, error) {
	backupDir := getString(params, "backup_dir")
	if backupDir == "" {
		return nil, fmt.Errorf("参数 backup_dir 是必需的")
	}
	dmIniPath := getString(params, "dm_ini_path")
	if dmIniPath == "" {
		return nil, fmt.Errorf("参数 dm_ini_path 是必需的")
	}

	dmrmanPath := firstNonEmpty(getString(params, "dmrman_path"), os.Getenv("DM_RMAN_PATH"), "dmrman")
	timeoutSeconds := getInt(params, "timeout_seconds", 600)
	backupName := firstNonEmpty(getString(params, "backup_name"), "full_backup")

	cfg := getConfigForBackup()

	script := fmt.Sprintf("BACKUP DATABASE '%s' BACKUPSET '%s/%s'",
		dmIniPath, backupDir, backupName)

	args := []string{"CTRLSCRIPT", script}

	output, err := runBackupCommand(dmrmanPath, args, timeoutSeconds, cfg.password)
	if err != nil {
		return nil, fmt.Errorf("物理备份失败: %w; output=%s", err, output)
	}

	return map[string]interface{}{
		"success":      true,
		"backup_dir":   backupDir,
		"backup_name":  backupName,
		"dm_ini_path":  dmIniPath,
		"output":       output,
	}, nil
}

func handlePhysicalRestore(params map[string]interface{}) (interface{}, error) {
	if !getBool(params, "confirm") {
		return nil, fmt.Errorf("物理恢复必须显式设置 confirm=true")
	}

	backupDir := getString(params, "backup_dir")
	if backupDir == "" {
		return nil, fmt.Errorf("参数 backup_dir 是必需的")
	}
	dmIniPath := getString(params, "dm_ini_path")
	if dmIniPath == "" {
		return nil, fmt.Errorf("参数 dm_ini_path 是必需的")
	}

	dmrmanPath := firstNonEmpty(getString(params, "dmrman_path"), os.Getenv("DM_RMAN_PATH"), "dmrman")
	timeoutSeconds := getInt(params, "timeout_seconds", 600)

	cfg := getConfigForBackup()

	script := fmt.Sprintf("RESTORE DATABASE '%s' FROM BACKUPSET '%s'",
		dmIniPath, backupDir)

	args := []string{"CTRLSCRIPT", script}

	output, err := runBackupCommand(dmrmanPath, args, timeoutSeconds, cfg.password)
	if err != nil {
		return nil, fmt.Errorf("物理恢复失败: %w; output=%s", err, output)
	}

	return map[string]interface{}{
		"success":     true,
		"backup_dir":  backupDir,
		"dm_ini_path": dmIniPath,
		"output":      output,
	}, nil
}

type backupConfig struct {
	user     string
	password string
	host     string
	port     int
}

func getConfigForBackup() backupConfig {
	return backupConfig{
		user:     getEnvOrDefault("DM_USER", "SYSDBA"),
		password: getEnvOrDefault("DM_PASSWORD", ""),
		host:     getEnvOrDefault("DM_HOST", "localhost"),
		port:     getIntFromString(getEnvOrDefault("DM_PORT", "5236"), 5236),
	}
}

func getIntFromString(s string, defaultVal int) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return defaultVal
	}
	return n
}

func runBackupCommand(binPath string, args []string, timeoutSeconds int, password string) (string, error) {
	ctx := context.Background()
	var cancel context.CancelFunc
	if timeoutSeconds > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()

	result := output.String()
	if password != "" {
		result = strings.ReplaceAll(result, password, "******")
	}
	result = truncateString(result, 4096)

	return result, err
}
