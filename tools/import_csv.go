package tools

import (
	"bytes"
	"context"
	"dm-mcp2/config"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultCSVDelimiter      = ","
	defaultCSVCharacterCode  = "UTF-8"
	defaultCSVRows           = 50000
	defaultCSVIndexOption    = 2
	defaultCSVMaxParallel    = 2
	defaultCSVMode           = "APPEND"
	defaultCSVOutputMaxBytes = 4096
)

var dmIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$#]*$`)

type csvImportOptions struct {
	Files          []csvImportFile
	DMFLDRPath     string
	WorkDir        string
	Delimiter      string
	EnclosedBy     string
	CharacterCode  string
	Rows           int
	Direct         bool
	IndexOption    int
	Mode           string
	MaxParallel    int
	TimeoutSeconds int
	Errors         int
}

type csvImportFile struct {
	CSVFile     string
	Schema      string
	Table       string
	Columns     []string
	ColumnTypes []string // 字段类型，如 CHAR, INTEGER, DATE, TIMESTAMP 等
	NullIf      string   // 空值处理，如 '' 或 BLANKS
	Skip        int
	BadFile     string
	LogFile     string
}

type csvImportResult struct {
	Index       int    `json:"index"`
	CSVFile     string `json:"csv_file"`
	Table       string `json:"table"`
	Status      string `json:"status"`
	DurationMS  int64  `json:"duration_ms"`
	ControlFile string `json:"control_file"`
	LogFile     string `json:"log_file"`
	BadFile     string `json:"bad_file"`
	ExitError   string `json:"exit_error,omitempty"`
	Output      string `json:"output,omitempty"`
}

type dmfldrRunner func(ctx context.Context, dmfldrPath string, args []string) (string, error)

func init() {
	registerImportTools()
}

func registerImportTools() {
	RegisterTool(ToolInfo{
		Name:        "batch_import_csv",
		Category:    "import",
		Description: "使用 DMFLDR 批量并行导入 CSV 文件。参数: files(必填)-数组[{csv_file,table,schema,columns,column_types,null_if,skip,bad_file,log_file}], dmfldr_path(可选), work_dir(可选), delimiter(可选,默认逗号), enclosed_by(可选), character_code(可选,默认UTF-8), rows(可选,默认50000), direct(可选,默认true), index_option(可选,默认2), mode(可选,APPEND|REPLACE|INSERT), max_parallel(可选,默认2), timeout_seconds(可选), errors(可选,默认0)",
		Params:      []string{"files", "dmfldr_path", "work_dir", "delimiter", "enclosed_by", "character_code", "rows", "direct", "index_option", "mode", "max_parallel", "timeout_seconds", "errors"},
	}, handleBatchImportCSV)
}

func handleBatchImportCSV(params map[string]interface{}) (interface{}, error) {
	options, err := parseCSVImportOptions(params)
	if err != nil {
		return nil, err
	}

	results := runCSVImports(options, defaultDMFLDRRunner)
	succeeded := 0
	for _, result := range results {
		if result.Status == "success" {
			succeeded++
		}
	}

	return map[string]interface{}{
		"success":      succeeded == len(results),
		"total":        len(results),
		"succeeded":    succeeded,
		"failed":       len(results) - succeeded,
		"max_parallel": options.MaxParallel,
		"results":      results,
	}, nil
}

func parseCSVImportOptions(params map[string]interface{}) (csvImportOptions, error) {
	options := csvImportOptions{
		DMFLDRPath:    firstNonEmpty(getString(params, "dmfldr_path"), os.Getenv("DM_FLDR_PATH"), "dmfldr"),
		WorkDir:       getString(params, "work_dir"),
		Delimiter:     firstNonEmpty(getString(params, "delimiter"), defaultCSVDelimiter),
		EnclosedBy:    getString(params, "enclosed_by"),
		CharacterCode: firstNonEmpty(getString(params, "character_code"), defaultCSVCharacterCode),
		Rows:          getInt(params, "rows", defaultCSVRows),
		Direct:        getBoolOrDefault(params, "direct", true),
		IndexOption:   getInt(params, "index_option", defaultCSVIndexOption),
		Mode:          strings.ToUpper(firstNonEmpty(getString(params, "mode"), defaultCSVMode)),
		MaxParallel:   getInt(params, "max_parallel", defaultCSVMaxParallel),
	}

	options.TimeoutSeconds = getInt(params, "timeout_seconds", 0)
	options.Errors = getInt(params, "errors", 0)

	if options.Rows <= 0 {
		return options, fmt.Errorf("参数 rows 必须大于 0")
	}
	if options.IndexOption < 1 || options.IndexOption > 3 {
		return options, fmt.Errorf("参数 index_option 必须是 1、2 或 3")
	}
	if options.MaxParallel <= 0 {
		return options, fmt.Errorf("参数 max_parallel 必须大于 0")
	}
	if options.TimeoutSeconds < 0 {
		return options, fmt.Errorf("参数 timeout_seconds 不能小于 0")
	}
	if !isValidCSVImportMode(options.Mode) {
		return options, fmt.Errorf("参数 mode 必须是 APPEND、REPLACE 或 INSERT")
	}
	if err := validateControlString("delimiter", options.Delimiter); err != nil {
		return options, err
	}
	if options.EnclosedBy != "" {
		if err := validateControlString("enclosed_by", options.EnclosedBy); err != nil {
			return options, err
		}
	}
	if err := validateControlString("character_code", options.CharacterCode); err != nil {
		return options, err
	}

	if options.WorkDir == "" {
		options.WorkDir = os.TempDir()
	}
	absWorkDir, err := filepath.Abs(options.WorkDir)
	if err != nil {
		return options, fmt.Errorf("解析 work_dir 失败: %w", err)
	}
	info, err := os.Stat(absWorkDir)
	if err != nil {
		return options, fmt.Errorf("work_dir 不可访问: %w", err)
	}
	if !info.IsDir() {
		return options, fmt.Errorf("work_dir 必须是目录")
	}
	options.WorkDir = absWorkDir

	files, ok := params["files"].([]interface{})
	if !ok || len(files) == 0 {
		return options, fmt.Errorf("参数 files 是必需的且不能为空")
	}

	options.Files = make([]csvImportFile, 0, len(files))
	for i, rawFile := range files {
		fileParams, ok := rawFile.(map[string]interface{})
		if !ok {
			return options, fmt.Errorf("files[%d] 必须是对象", i)
		}
		file, err := parseCSVImportFile(i, fileParams)
		if err != nil {
			return options, err
		}
		options.Files = append(options.Files, file)
	}

	if options.MaxParallel > len(options.Files) {
		options.MaxParallel = len(options.Files)
	}

	return options, nil
}

func parseCSVImportFile(index int, params map[string]interface{}) (csvImportFile, error) {
	file := csvImportFile{
		CSVFile: getString(params, "csv_file"),
		Schema:  getString(params, "schema"),
		Table:   getString(params, "table"),
		NullIf:  getString(params, "null_if"),
		Skip:    getInt(params, "skip", 0),
		BadFile: getString(params, "bad_file"),
		LogFile: getString(params, "log_file"),
	}

	if file.CSVFile == "" {
		return file, fmt.Errorf("files[%d].csv_file 是必需的", index)
	}
	absCSVFile, err := filepath.Abs(file.CSVFile)
	if err != nil {
		return file, fmt.Errorf("解析 files[%d].csv_file 失败: %w", index, err)
	}
	info, err := os.Stat(absCSVFile)
	if err != nil {
		return file, fmt.Errorf("files[%d].csv_file 不可访问: %w", index, err)
	}
	if info.IsDir() {
		return file, fmt.Errorf("files[%d].csv_file 必须是文件", index)
	}
	file.CSVFile = absCSVFile

	if file.Table == "" {
		return file, fmt.Errorf("files[%d].table 是必需的", index)
	}
	if strings.Contains(file.Table, ".") {
		if file.Schema != "" {
			return file, fmt.Errorf("files[%d] 不能同时设置 schema 和带 schema 的 table", index)
		}
		parts := strings.Split(file.Table, ".")
		if len(parts) != 2 {
			return file, fmt.Errorf("files[%d].table 格式必须是 TABLE 或 SCHEMA.TABLE", index)
		}
		file.Schema = parts[0]
		file.Table = parts[1]
	}
	if err := validateIdentifier("table", file.Table); err != nil {
		return file, fmt.Errorf("files[%d].%w", index, err)
	}
	if file.Schema != "" {
		if err := validateIdentifier("schema", file.Schema); err != nil {
			return file, fmt.Errorf("files[%d].%w", index, err)
		}
	}
	if file.Skip < 0 {
		return file, fmt.Errorf("files[%d].skip 不能小于 0", index)
	}

	columns, err := parseStringArray(params, "columns")
	if err != nil {
		return file, fmt.Errorf("files[%d].%w", index, err)
	}
	for _, column := range columns {
		if err := validateIdentifier("columns", column); err != nil {
			return file, fmt.Errorf("files[%d].%w", index, err)
		}
	}
	file.Columns = columns

	columnTypes, err := parseStringArray(params, "column_types")
	if err != nil {
		return file, fmt.Errorf("files[%d].%w", index, err)
	}
	file.ColumnTypes = columnTypes

	if file.BadFile != "" {
		file.BadFile, err = filepath.Abs(file.BadFile)
		if err != nil {
			return file, fmt.Errorf("解析 files[%d].bad_file 失败: %w", index, err)
		}
	}
	if file.LogFile != "" {
		file.LogFile, err = filepath.Abs(file.LogFile)
		if err != nil {
			return file, fmt.Errorf("解析 files[%d].log_file 失败: %w", index, err)
		}
	}

	return file, nil
}

func runCSVImports(options csvImportOptions, runner dmfldrRunner) []csvImportResult {
	results := make([]csvImportResult, len(options.Files))
	sem := make(chan struct{}, options.MaxParallel)
	var wg sync.WaitGroup

	for i := range options.Files {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = runSingleCSVImport(i, options, runner)
		}()
	}

	wg.Wait()
	return results
}

func runSingleCSVImport(index int, options csvImportOptions, runner dmfldrRunner) (result csvImportResult) {
	start := time.Now()
	file := options.Files[index]
	result = csvImportResult{
		Index:   index,
		CSVFile: file.CSVFile,
		Table:   qualifiedTableName(file.Schema, file.Table),
		Status:  "failed",
	}
	defer func() {
		result.DurationMS = time.Since(start).Milliseconds()
	}()

	paths, err := prepareCSVImportPaths(index, options.WorkDir, file)
	if err != nil {
		result.ExitError = err.Error()
		return result
	}
	result.ControlFile = paths.controlFile
	result.LogFile = paths.logFile
	result.BadFile = paths.badFile

	controlContent := buildCSVControlFile(options, file, paths.badFile)
	if err := os.WriteFile(paths.controlFile, []byte(controlContent), 0600); err != nil {
		result.ExitError = fmt.Sprintf("写入控制文件失败: %v", err)
		return result
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if options.TimeoutSeconds > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(options.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	args := buildDMFLDRArgs(paths.controlFile, paths.logFile)
	output, err := runner(ctx, options.DMFLDRPath, args)
	result.Output = truncateString(redactDMFLDROutput(output), defaultCSVOutputMaxBytes)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.ExitError = fmt.Sprintf("导入超时（%d 秒）", options.TimeoutSeconds)
		} else {
			result.ExitError = err.Error()
		}
		return result
	}

	result.Status = "success"
	return result
}

type csvImportPaths struct {
	controlFile string
	logFile     string
	badFile     string
}

func prepareCSVImportPaths(index int, workDir string, file csvImportFile) (csvImportPaths, error) {
	base := sanitizeFileBase(qualifiedTableName(file.Schema, file.Table))
	if base == "" {
		base = "csv_import"
	}
	base = fmt.Sprintf("%s_%d_%d", base, time.Now().UnixNano(), index)

	paths := csvImportPaths{
		controlFile: filepath.Join(workDir, base+".ctl"),
		logFile:     firstNonEmpty(file.LogFile, filepath.Join(workDir, base+".log")),
		badFile:     firstNonEmpty(file.BadFile, filepath.Join(workDir, base+".bad")),
	}

	for name, path := range map[string]string{
		"control_file": paths.controlFile,
		"log_file":     paths.logFile,
		"bad_file":     paths.badFile,
	} {
		if err := ensureParentDir(path); err != nil {
			return paths, fmt.Errorf("%s 父目录不可用: %w", name, err)
		}
	}

	return paths, nil
}

func buildCSVControlFile(options csvImportOptions, file csvImportFile, badFile string) string {
	var sb strings.Builder
	sb.WriteString("OPTIONS\n(\n")
	sb.WriteString("SKIP = ")
	sb.WriteString(strconv.Itoa(file.Skip))
	sb.WriteString("\nROWS = ")
	sb.WriteString(strconv.Itoa(options.Rows))
	sb.WriteString("\nDIRECT = ")
	sb.WriteString(formatDMBool(options.Direct))
	sb.WriteString("\nINDEX_OPTION = ")
	sb.WriteString(strconv.Itoa(options.IndexOption))
	sb.WriteString("\nCHARACTER_CODE=")
	sb.WriteString(quoteControlString(options.CharacterCode))
	if options.Errors > 0 {
		sb.WriteString("\nERRORS = ")
		sb.WriteString(strconv.Itoa(options.Errors))
	}
	sb.WriteString("\n)\n")
	sb.WriteString("LOAD DATA\n")
	sb.WriteString("INFILE ")
	sb.WriteString(quoteControlString(filepath.ToSlash(file.CSVFile)))
	sb.WriteString("\nBADFILE ")
	sb.WriteString(quoteControlString(filepath.ToSlash(badFile)))
	sb.WriteString("\n")
	sb.WriteString(options.Mode)
	sb.WriteString("\nINTO TABLE ")
	sb.WriteString(qualifiedTableName(file.Schema, file.Table))
	sb.WriteString("\nFIELDS TERMINATED BY ")
	sb.WriteString(quoteControlString(options.Delimiter))
	if options.EnclosedBy != "" {
		sb.WriteString("\nOPTIONALLY ENCLOSED BY ")
		sb.WriteString(quoteControlString(options.EnclosedBy))
	}
	if len(file.Columns) > 0 {
		sb.WriteString("\n(\n")
		for i, col := range file.Columns {
			if i > 0 {
				sb.WriteString(",\n")
			}
			sb.WriteString("  ")
			sb.WriteString(col)
			// 添加字段类型
			if i < len(file.ColumnTypes) && file.ColumnTypes[i] != "" {
				sb.WriteString(" ")
				sb.WriteString(file.ColumnTypes[i])
			}
			// 添加NULLIF处理
			if file.NullIf != "" {
				sb.WriteString(" NULLIF ")
				sb.WriteString(col)
				sb.WriteString(" = ")
				sb.WriteString(quoteControlString(file.NullIf))
			}
		}
		sb.WriteString("\n)")
	}
	sb.WriteString("\n")
	return sb.String()
}

func buildDMFLDRArgs(controlFile, logFile string) []string {
	cfg := config.LoadConfig()
	userID := fmt.Sprintf("userid=%s/%s@%s:%d", cfg.User, cfg.Password, cfg.Host, cfg.Port)
	return []string{
		userID,
		"control=" + controlFile,
		"log=" + logFile,
	}
}

func defaultDMFLDRRunner(ctx context.Context, dmfldrPath string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, dmfldrPath, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.String(), err
}

func parseStringArray(params map[string]interface{}, key string) ([]string, error) {
	raw, exists := params[key]
	if !exists || raw == nil {
		return nil, nil
	}
	rawItems, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("%s 必须是字符串数组", key)
	}
	items := make([]string, 0, len(rawItems))
	for i, rawItem := range rawItems {
		item, ok := rawItem.(string)
		if !ok || item == "" {
			return nil, fmt.Errorf("%s[%d] 必须是非空字符串", key, i)
		}
		items = append(items, item)
	}
	return items, nil
}

func validateIdentifier(name, value string) error {
	if !dmIdentifierPattern.MatchString(value) {
		return fmt.Errorf("%s 只能包含字母、数字、下划线、$、#，且必须以字母或下划线开头", name)
	}
	return nil
}

func validateControlString(name, value string) error {
	if value == "" {
		return fmt.Errorf("参数 %s 不能为空", name)
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("参数 %s 不能包含换行符", name)
	}
	return nil
}

func isValidCSVImportMode(mode string) bool {
	switch mode {
	case "APPEND", "REPLACE", "INSERT":
		return true
	default:
		return false
	}
}

func qualifiedTableName(schema, table string) string {
	if schema == "" {
		return table
	}
	return schema + "." + table
}

func formatDMBool(value bool) string {
	if value {
		return "TRUE"
	}
	return "FALSE"
}

func quoteControlString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s 不是目录", dir)
	}
	return nil
}

func sanitizeFileBase(value string) string {
	var sb strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			sb.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			sb.WriteRune(r)
		case r >= '0' && r <= '9':
			sb.WriteRune(r)
		case r == '_', r == '-', r == '.':
			sb.WriteRune(r)
		default:
			sb.WriteRune('_')
		}
	}
	return strings.Trim(sb.String(), "._-")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func truncateString(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes] + "...(truncated)"
}

func redactDMFLDROutput(output string) string {
	cfg := config.LoadConfig()
	for _, secret := range []string{cfg.Password, os.Getenv("DM_PASSWORD")} {
		if secret != "" {
			output = strings.ReplaceAll(output, secret, "******")
		}
	}
	return output
}
