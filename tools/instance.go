package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	defaultDminitPath       = "dminit"
	defaultDMDBName         = "DAMENG"
	defaultDMInstanceName   = "DMSERVER"
	defaultCommandOutputMax = 4096
)

type instanceCommandRunner func(ctx context.Context, name string, args []string) (string, error)

type createDatabaseOptions struct {
	DminitPath     string
	Path           string
	SYSDBAPwd      string
	SYSAuditorPwd  string
	DBName         string
	InstanceName   string
	PortNum        int
	PageSize       int
	ExtentSize     int
	CaseSensitive  string
	Charset        int
	LogSize        int
	TimeZone       string
	ExtraArgs      []string
	TimeoutSeconds int
}

type serviceControlOptions struct {
	InstanceName      string
	ServiceName       string
	BinDir            string
	ServiceScriptPath string
	ServiceManager    string
	OS                string
	Action            string
	TimeoutSeconds    int
}

type serviceControlResult struct {
	Success        bool     `json:"success"`
	Action         string   `json:"action"`
	ServiceName    string   `json:"service_name"`
	ServiceManager string   `json:"service_manager"`
	Command        string   `json:"command"`
	Args           []string `json:"args"`
	Output         string   `json:"output,omitempty"`
}

type deleteDatabaseOptions struct {
	DatabaseDir    string
	Confirm        bool
	StopService    bool
	InstanceName   string
	ServiceName    string
	BinDir         string
	ServiceManager string
	TimeoutSeconds int
}

type deleteDatabaseResult struct {
	Success        bool                  `json:"success"`
	DatabaseDir    string                `json:"database_dir"`
	ServiceName    string                `json:"service_name,omitempty"`
	ServiceStopped bool                  `json:"service_stopped"`
	StopResult     *serviceControlResult `json:"stop_result,omitempty"`
}

func init() {
	registerInstanceTools()
}

func registerInstanceTools() {
	RegisterTool(ToolInfo{
		Name:        "create_database",
		Category:    "instance",
		Description: "使用 dminit 初始化达梦数据库实例。参数: path, sysdba_pwd, sysauditor_pwd; 可选 dminit_path, db_name, instance_name, port_num, page_size, extent_size, case_sensitive, charset, log_size, time_zone, extra_args, timeout_seconds",
		Params:      []string{"path", "sysdba_pwd", "sysauditor_pwd", "dminit_path", "db_name", "instance_name", "port_num", "page_size", "extent_size", "case_sensitive", "charset", "log_size", "time_zone", "extra_args", "timeout_seconds"},
	}, handleCreateDatabase)

	RegisterTool(ToolInfo{
		Name:        "delete_database",
		Category:    "instance",
		Description: "强确认删除达梦数据库实例目录。参数: database_dir, confirm=true; 可选 instance_name, service_name, stop_service(默认true), timeout_seconds",
		Params:      []string{"database_dir", "confirm", "instance_name", "service_name", "stop_service", "timeout_seconds"},
	}, handleDeleteDatabase)

	RegisterTool(ToolInfo{
		Name:        "start_database_service",
		Category:    "instance",
		Description: "启动达梦数据库服务。参数: instance_name 或 service_name; 可选 bin_dir, service_script_path, service_manager(auto|script|systemd|windows), timeout_seconds",
		Params:      []string{"instance_name", "service_name", "bin_dir", "service_script_path", "service_manager", "timeout_seconds"},
	}, func(params map[string]interface{}) (interface{}, error) {
		return handleDatabaseServiceAction(params, "start")
	})

	RegisterTool(ToolInfo{
		Name:        "stop_database_service",
		Category:    "instance",
		Description: "停止达梦数据库服务。参数: instance_name 或 service_name; 可选 bin_dir, service_script_path, service_manager(auto|script|systemd|windows), timeout_seconds",
		Params:      []string{"instance_name", "service_name", "bin_dir", "service_script_path", "service_manager", "timeout_seconds"},
	}, func(params map[string]interface{}) (interface{}, error) {
		return handleDatabaseServiceAction(params, "stop")
	})

	RegisterTool(ToolInfo{
		Name:        "restart_database_service",
		Category:    "instance",
		Description: "重启达梦数据库服务。参数: instance_name 或 service_name; 可选 bin_dir, service_script_path, service_manager(auto|script|systemd|windows), timeout_seconds",
		Params:      []string{"instance_name", "service_name", "bin_dir", "service_script_path", "service_manager", "timeout_seconds"},
	}, func(params map[string]interface{}) (interface{}, error) {
		return handleDatabaseServiceAction(params, "restart")
	})

	RegisterTool(ToolInfo{
		Name:        "database_service_status",
		Category:    "instance",
		Description: "查看达梦数据库服务状态。参数: instance_name 或 service_name; 可选 bin_dir, service_script_path, service_manager(auto|script|systemd|windows), timeout_seconds",
		Params:      []string{"instance_name", "service_name", "bin_dir", "service_script_path", "service_manager", "timeout_seconds"},
	}, func(params map[string]interface{}) (interface{}, error) {
		return handleDatabaseServiceAction(params, "status")
	})
}

func handleCreateDatabase(params map[string]interface{}) (interface{}, error) {
	options, err := parseCreateDatabaseOptions(params)
	if err != nil {
		return nil, err
	}
	return createDatabaseInstance(options, defaultInstanceCommandRunner)
}

func handleDeleteDatabase(params map[string]interface{}) (interface{}, error) {
	options, err := parseDeleteDatabaseOptions(params)
	if err != nil {
		return nil, err
	}
	return deleteDatabaseInstance(options, defaultInstanceCommandRunner)
}

func handleDatabaseServiceAction(params map[string]interface{}, action string) (interface{}, error) {
	options, err := parseServiceControlOptions(params, action)
	if err != nil {
		return nil, err
	}
	return runDatabaseService(options, defaultInstanceCommandRunner)
}

func parseCreateDatabaseOptions(params map[string]interface{}) (createDatabaseOptions, error) {
	options := createDatabaseOptions{
		DminitPath:     firstNonEmpty(getString(params, "dminit_path"), defaultDminitPath),
		Path:           strings.TrimSpace(getString(params, "path")),
		SYSDBAPwd:      getString(params, "sysdba_pwd"),
		SYSAuditorPwd:  getString(params, "sysauditor_pwd"),
		DBName:         strings.TrimSpace(getString(params, "db_name")),
		InstanceName:   strings.TrimSpace(getString(params, "instance_name")),
		PortNum:        getInt(params, "port_num", 0),
		PageSize:       getInt(params, "page_size", 0),
		ExtentSize:     getInt(params, "extent_size", 0),
		CaseSensitive:  strings.TrimSpace(getString(params, "case_sensitive")),
		Charset:        getInt(params, "charset", -1),
		LogSize:        getInt(params, "log_size", 0),
		TimeZone:       strings.TrimSpace(getString(params, "time_zone")),
		TimeoutSeconds: getInt(params, "timeout_seconds", 0),
	}

	if options.Path == "" {
		return options, fmt.Errorf("参数 path 是必需的")
	}
	if options.SYSDBAPwd == "" {
		return options, fmt.Errorf("参数 sysdba_pwd 是必需的")
	}
	if options.SYSAuditorPwd == "" {
		return options, fmt.Errorf("参数 sysauditor_pwd 是必需的")
	}
	if options.TimeoutSeconds < 0 {
		return options, fmt.Errorf("参数 timeout_seconds 不能小于 0")
	}
	if err := validateOptionalDMIdentifier("db_name", options.DBName); err != nil {
		return options, err
	}
	if err := validateOptionalDMIdentifier("instance_name", options.InstanceName); err != nil {
		return options, err
	}
	if options.PortNum != 0 && (options.PortNum < 1024 || options.PortNum > 65534) {
		return options, fmt.Errorf("参数 port_num 必须在 1024 到 65534 之间")
	}
	if options.CaseSensitive != "" && options.CaseSensitive != "Y" && options.CaseSensitive != "N" && options.CaseSensitive != "1" && options.CaseSensitive != "0" {
		return options, fmt.Errorf("参数 case_sensitive 必须是 Y、N、1 或 0")
	}

	extraArgs, err := parseStringArray(params, "extra_args")
	if err != nil {
		return options, err
	}
	for i, arg := range extraArgs {
		if strings.ContainsAny(arg, "\r\n") || !strings.Contains(arg, "=") {
			return options, fmt.Errorf("extra_args[%d] 必须是单行 KEY=VALUE 格式", i)
		}
	}
	options.ExtraArgs = extraArgs

	return options, nil
}

func buildDminitArgs(options createDatabaseOptions) []string {
	args := []string{
		"PATH=" + options.Path,
		"SYSDBA_PWD=" + options.SYSDBAPwd,
		"SYSAUDITOR_PWD=" + options.SYSAuditorPwd,
	}
	if options.DBName != "" {
		args = append(args, "DB_NAME="+options.DBName)
	}
	if options.InstanceName != "" {
		args = append(args, "INSTANCE_NAME="+options.InstanceName)
	}
	if options.PortNum > 0 {
		args = append(args, fmt.Sprintf("PORT_NUM=%d", options.PortNum))
	}
	if options.PageSize > 0 {
		args = append(args, fmt.Sprintf("PAGE_SIZE=%d", options.PageSize))
	}
	if options.ExtentSize > 0 {
		args = append(args, fmt.Sprintf("EXTENT_SIZE=%d", options.ExtentSize))
	}
	if options.CaseSensitive != "" {
		args = append(args, "CASE_SENSITIVE="+options.CaseSensitive)
	}
	if options.Charset >= 0 {
		args = append(args, fmt.Sprintf("CHARSET=%d", options.Charset))
	}
	if options.LogSize > 0 {
		args = append(args, fmt.Sprintf("LOG_SIZE=%d", options.LogSize))
	}
	if options.TimeZone != "" {
		args = append(args, "TIME_ZONE="+options.TimeZone)
	}
	args = append(args, options.ExtraArgs...)
	return args
}

func createDatabaseInstance(options createDatabaseOptions, runner instanceCommandRunner) (map[string]interface{}, error) {
	ctx, cancel := contextForTimeout(options.TimeoutSeconds)
	defer cancel()

	args := buildDminitArgs(options)
	output, err := runner(ctx, options.DminitPath, args)
	redactedOutput := truncateString(redactInstanceOutput(output, options), defaultCommandOutputMax)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("初始化数据库超时（%d 秒）: %s", options.TimeoutSeconds, redactedOutput)
		}
		return nil, fmt.Errorf("初始化数据库失败: %w; output=%s", err, redactedOutput)
	}

	dbName := firstNonEmpty(options.DBName, defaultDMDBName)
	instanceName := firstNonEmpty(options.InstanceName, defaultDMInstanceName)
	dbDir := filepath.Join(options.Path, dbName)

	return map[string]interface{}{
		"success":      true,
		"output":       redactedOutput,
		"db_dir":       dbDir,
		"dm_ini_path":  filepath.Join(dbDir, "dm.ini"),
		"service_name": serviceNameForInstance(instanceName),
		"dminit_args":  redactDminitArgs(args, options),
	}, nil
}

func redactInstanceOutput(output string, options createDatabaseOptions) string {
	redacted := output
	for _, secret := range []string{options.SYSDBAPwd, options.SYSAuditorPwd} {
		if secret != "" {
			redacted = strings.ReplaceAll(redacted, secret, "******")
		}
	}
	return redacted
}

func redactDminitArgs(args []string, options createDatabaseOptions) []string {
	redacted := make([]string, len(args))
	for i, arg := range args {
		upper := strings.ToUpper(arg)
		if strings.HasPrefix(upper, "SYSDBA_PWD=") || strings.HasPrefix(upper, "SYSAUDITOR_PWD=") || strings.Contains(upper, "_PWD=") {
			key, _, _ := strings.Cut(arg, "=")
			redacted[i] = key + "=******"
			continue
		}
		redacted[i] = arg
	}
	return redacted
}

func parseServiceControlOptions(params map[string]interface{}, action string) (serviceControlOptions, error) {
	options := serviceControlOptions{
		InstanceName:      strings.TrimSpace(getString(params, "instance_name")),
		ServiceName:       strings.TrimSpace(getString(params, "service_name")),
		BinDir:            strings.TrimSpace(getString(params, "bin_dir")),
		ServiceScriptPath: strings.TrimSpace(getString(params, "service_script_path")),
		ServiceManager:    strings.ToLower(strings.TrimSpace(firstNonEmpty(getString(params, "service_manager"), "auto"))),
		OS:                runtime.GOOS,
		Action:            action,
		TimeoutSeconds:    getInt(params, "timeout_seconds", 0),
	}

	if options.ServiceName == "" {
		if options.InstanceName == "" {
			return options, fmt.Errorf("参数 instance_name 或 service_name 至少需要一个")
		}
		if err := validateIdentifier("instance_name", options.InstanceName); err != nil {
			return options, err
		}
		options.ServiceName = serviceNameForInstance(options.InstanceName)
	}
	if options.TimeoutSeconds < 0 {
		return options, fmt.Errorf("参数 timeout_seconds 不能小于 0")
	}
	if !isValidServiceAction(action) {
		return options, fmt.Errorf("不支持的服务操作: %s", action)
	}
	if !isValidServiceManager(options.ServiceManager) {
		return options, fmt.Errorf("service_manager 必须是 auto、script、systemd 或 windows")
	}
	return options, nil
}

func runDatabaseService(options serviceControlOptions, runner instanceCommandRunner) (serviceControlResult, error) {
	if options.ServiceName == "" && options.InstanceName != "" {
		options.ServiceName = serviceNameForInstance(options.InstanceName)
	}

	name, args, manager, err := buildServiceCommand(options)
	if err != nil {
		return serviceControlResult{}, err
	}

	ctx, cancel := contextForTimeout(options.TimeoutSeconds)
	defer cancel()

	if manager == "windows" && options.Action == "restart" {
		return runWindowsServiceRestart(ctx, options, runner)
	}

	output, err := runner(ctx, name, args)
	result := serviceControlResult{
		Success:        err == nil,
		Action:         options.Action,
		ServiceName:    options.ServiceName,
		ServiceManager: manager,
		Command:        name,
		Args:           args,
		Output:         truncateString(output, defaultCommandOutputMax),
	}
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return result, fmt.Errorf("服务操作超时（%d 秒）", options.TimeoutSeconds)
		}
		return result, err
	}
	return result, nil
}

func runWindowsServiceRestart(ctx context.Context, options serviceControlOptions, runner instanceCommandRunner) (serviceControlResult, error) {
	outputParts := make([]string, 0, 2)
	for _, action := range []string{"stop", "start"} {
		output, err := runner(ctx, "sc.exe", []string{action, options.ServiceName})
		outputParts = append(outputParts, output)
		if err != nil {
			return serviceControlResult{
				Success:        false,
				Action:         options.Action,
				ServiceName:    options.ServiceName,
				ServiceManager: "windows",
				Command:        "sc.exe",
				Args:           []string{action, options.ServiceName},
				Output:         truncateString(strings.Join(outputParts, "\n"), defaultCommandOutputMax),
			}, err
		}
	}
	return serviceControlResult{
		Success:        true,
		Action:         options.Action,
		ServiceName:    options.ServiceName,
		ServiceManager: "windows",
		Command:        "sc.exe",
		Args:           []string{"stop/start", options.ServiceName},
		Output:         truncateString(strings.Join(outputParts, "\n"), defaultCommandOutputMax),
	}, nil
}

func buildServiceCommand(options serviceControlOptions) (string, []string, string, error) {
	manager := options.ServiceManager
	if manager == "" || manager == "auto" {
		switch {
		case strings.EqualFold(options.OS, "windows"):
			manager = "windows"
		case options.ServiceScriptPath != "" || options.BinDir != "":
			manager = "script"
		default:
			manager = "systemd"
		}
	}

	switch manager {
	case "windows":
		scAction := options.Action
		if options.Action == "status" {
			scAction = "query"
		}
		return "sc.exe", []string{scAction, options.ServiceName}, manager, nil
	case "script":
		scriptPath := options.ServiceScriptPath
		if scriptPath == "" {
			if options.BinDir == "" {
				return "", nil, "", fmt.Errorf("service_manager=script 时需要 service_script_path 或 bin_dir")
			}
			scriptPath = filepath.Join(options.BinDir, options.ServiceName)
		}
		return scriptPath, []string{scriptAction(options.Action)}, manager, nil
	case "systemd":
		unit := options.ServiceName
		if !strings.HasSuffix(unit, ".service") {
			unit += ".service"
		}
		return "systemctl", []string{systemdAction(options.Action), unit}, manager, nil
	default:
		return "", nil, "", fmt.Errorf("service_manager 必须是 auto、script、systemd 或 windows")
	}
}

func parseDeleteDatabaseOptions(params map[string]interface{}) (deleteDatabaseOptions, error) {
	options := deleteDatabaseOptions{
		DatabaseDir:    strings.TrimSpace(getString(params, "database_dir")),
		Confirm:        getBool(params, "confirm"),
		StopService:    getBoolOrDefault(params, "stop_service", true),
		InstanceName:   strings.TrimSpace(getString(params, "instance_name")),
		ServiceName:    strings.TrimSpace(getString(params, "service_name")),
		BinDir:         strings.TrimSpace(getString(params, "bin_dir")),
		ServiceManager: strings.ToLower(strings.TrimSpace(firstNonEmpty(getString(params, "service_manager"), "auto"))),
		TimeoutSeconds: getInt(params, "timeout_seconds", 0),
	}

	if options.DatabaseDir == "" {
		return options, fmt.Errorf("参数 database_dir 是必需的")
	}
	if !options.Confirm {
		return options, fmt.Errorf("删除数据库必须显式设置 confirm=true")
	}
	if options.TimeoutSeconds < 0 {
		return options, fmt.Errorf("参数 timeout_seconds 不能小于 0")
	}

	absDir, err := filepath.Abs(options.DatabaseDir)
	if err != nil {
		return options, fmt.Errorf("解析 database_dir 失败: %w", err)
	}
	info, err := os.Stat(absDir)
	if err != nil {
		return options, fmt.Errorf("database_dir 不可访问: %w", err)
	}
	if !info.IsDir() {
		return options, fmt.Errorf("database_dir 必须是目录")
	}
	if isFilesystemRoot(absDir) {
		return options, fmt.Errorf("database_dir 不能是文件系统根目录")
	}
	if _, err := os.Stat(filepath.Join(absDir, "dm.ini")); err != nil {
		return options, fmt.Errorf("database_dir 必须包含 dm.ini 以确认这是达梦实例目录: %w", err)
	}
	options.DatabaseDir = absDir

	if options.ServiceName == "" && options.InstanceName != "" {
		if err := validateIdentifier("instance_name", options.InstanceName); err != nil {
			return options, err
		}
		options.ServiceName = serviceNameForInstance(options.InstanceName)
	}
	if options.ServiceManager != "" && !isValidServiceManager(options.ServiceManager) {
		return options, fmt.Errorf("service_manager 必须是 auto、script、systemd 或 windows")
	}
	return options, nil
}

func deleteDatabaseInstance(options deleteDatabaseOptions, runner instanceCommandRunner) (deleteDatabaseResult, error) {
	result := deleteDatabaseResult{
		Success:     false,
		DatabaseDir: options.DatabaseDir,
		ServiceName: options.ServiceName,
	}

	if options.StopService && options.ServiceName != "" {
		stopResult, err := runDatabaseService(serviceControlOptions{
			InstanceName:   options.InstanceName,
			ServiceName:    options.ServiceName,
			BinDir:         options.BinDir,
			ServiceManager: options.ServiceManager,
			OS:             runtime.GOOS,
			Action:         "stop",
			TimeoutSeconds: options.TimeoutSeconds,
		}, runner)
		result.StopResult = &stopResult
		if err != nil {
			return result, err
		}
		result.ServiceStopped = true
	}

	if err := os.RemoveAll(options.DatabaseDir); err != nil {
		return result, fmt.Errorf("删除数据库目录失败: %w", err)
	}
	result.Success = true
	return result, nil
}

func defaultInstanceCommandRunner(ctx context.Context, name string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.String(), err
}

func contextForTimeout(timeoutSeconds int) (context.Context, context.CancelFunc) {
	if timeoutSeconds > 0 {
		return context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	}
	return context.WithCancel(context.Background())
}

func serviceNameForInstance(instanceName string) string {
	if strings.HasPrefix(instanceName, "DmService") {
		return instanceName
	}
	return "DmService" + instanceName
}

func isValidServiceAction(action string) bool {
	switch action {
	case "start", "stop", "restart", "status":
		return true
	default:
		return false
	}
}

func isValidServiceManager(manager string) bool {
	switch manager {
	case "", "auto", "script", "systemd", "windows":
		return true
	default:
		return false
	}
}

func scriptAction(action string) string {
	if action == "status" {
		return "status"
	}
	return action
}

func systemdAction(action string) string {
	if action == "status" {
		return "status"
	}
	return action
}

func validateOptionalDMIdentifier(name, value string) error {
	if value == "" {
		return nil
	}
	return validateIdentifier(name, value)
}

func isFilesystemRoot(path string) bool {
	clean := filepath.Clean(path)
	parent := filepath.Dir(clean)
	return clean == parent
}
