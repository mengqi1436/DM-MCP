package tools

import (
	"fmt"
)

func init() {
	registerAdminExtendedTools()
}

func registerAdminExtendedTools() {
	RegisterTool(ToolInfo{
		Name:        "create_user",
		Category:    "admin",
		Description: "创建数据库用户。参数: username(必填), password(必填), default_tablespace(可选)",
		Params:      []string{"username", "password", "default_tablespace"},
	}, handleCreateUser)

	RegisterTool(ToolInfo{
		Name:        "drop_user",
		Category:    "admin",
		Description: "删除数据库用户。参数: username(必填), cascade(可选)-同时删除其对象",
		Params:      []string{"username", "cascade"},
	}, handleDropUser)

	RegisterTool(ToolInfo{
		Name:        "grant_privilege",
		Category:    "admin",
		Description: "授予权限。参数: privilege(必填)-权限名(如DBA,RESOURCE,SELECT ANY TABLE), grantee(必填)-用户或角色名",
		Params:      []string{"privilege", "grantee"},
	}, handleGrantPrivilege)

	RegisterTool(ToolInfo{
		Name:        "revoke_privilege",
		Category:    "admin",
		Description: "撤销权限。参数: privilege(必填), grantee(必填)-用户或角色名",
		Params:      []string{"privilege", "grantee"},
	}, handleRevokePrivilege)

	RegisterTool(ToolInfo{
		Name:        "create_role",
		Category:    "admin",
		Description: "创建角色。参数: role_name(必填)",
		Params:      []string{"role_name"},
	}, handleCreateRole)

	RegisterTool(ToolInfo{
		Name:        "drop_role",
		Category:    "admin",
		Description: "删除角色。参数: role_name(必填)",
		Params:      []string{"role_name"},
	}, handleDropRole)

	RegisterTool(ToolInfo{
		Name:        "list_roles",
		Category:    "admin",
		Description: "列出所有角色",
		Params:      []string{},
	}, handleListRoles)

	RegisterTool(ToolInfo{
		Name:        "list_tablespaces",
		Category:    "admin",
		Description: "列出所有表空间及使用信息",
		Params:      []string{},
	}, handleListTablespaces)

	RegisterTool(ToolInfo{
		Name:        "create_tablespace",
		Category:    "admin",
		Description: "创建表空间。参数: tablespace_name(必填), datafile(必填)-数据文件路径, size(可选,默认128MB)",
		Params:      []string{"tablespace_name", "datafile", "size"},
	}, handleCreateTablespace)
}

func handleCreateUser(params map[string]interface{}) (interface{}, error) {
	username := getString(params, "username")
	if username == "" {
		return nil, fmt.Errorf("参数 username 是必需的")
	}
	password := getString(params, "password")
	if password == "" {
		return nil, fmt.Errorf("参数 password 是必需的")
	}

	sql := fmt.Sprintf("CREATE USER %s IDENTIFIED BY \"%s\"", username, password)
	if ts := getString(params, "default_tablespace"); ts != "" {
		sql += " DEFAULT TABLESPACE " + ts
	}

	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "message": fmt.Sprintf("用户 %s 创建成功", username)}, nil
}

func handleDropUser(params map[string]interface{}) (interface{}, error) {
	username := getString(params, "username")
	if username == "" {
		return nil, fmt.Errorf("参数 username 是必需的")
	}

	sql := "DROP USER " + username
	if getBool(params, "cascade") {
		sql += " CASCADE"
	}

	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "message": fmt.Sprintf("用户 %s 删除成功", username)}, nil
}

func handleGrantPrivilege(params map[string]interface{}) (interface{}, error) {
	privilege := getString(params, "privilege")
	if privilege == "" {
		return nil, fmt.Errorf("参数 privilege 是必需的")
	}
	grantee := getString(params, "grantee")
	if grantee == "" {
		return nil, fmt.Errorf("参数 grantee 是必需的")
	}

	sql := fmt.Sprintf("GRANT %s TO %s", privilege, grantee)
	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "message": fmt.Sprintf("已将 %s 授予 %s", privilege, grantee)}, nil
}

func handleRevokePrivilege(params map[string]interface{}) (interface{}, error) {
	privilege := getString(params, "privilege")
	if privilege == "" {
		return nil, fmt.Errorf("参数 privilege 是必需的")
	}
	grantee := getString(params, "grantee")
	if grantee == "" {
		return nil, fmt.Errorf("参数 grantee 是必需的")
	}

	sql := fmt.Sprintf("REVOKE %s FROM %s", privilege, grantee)
	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "message": fmt.Sprintf("已从 %s 撤销 %s", grantee, privilege)}, nil
}

func handleCreateRole(params map[string]interface{}) (interface{}, error) {
	roleName := getString(params, "role_name")
	if roleName == "" {
		return nil, fmt.Errorf("参数 role_name 是必需的")
	}

	sql := "CREATE ROLE " + roleName
	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "message": fmt.Sprintf("角色 %s 创建成功", roleName)}, nil
}

func handleDropRole(params map[string]interface{}) (interface{}, error) {
	roleName := getString(params, "role_name")
	if roleName == "" {
		return nil, fmt.Errorf("参数 role_name 是必需的")
	}

	sql := "DROP ROLE " + roleName
	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "message": fmt.Sprintf("角色 %s 删除成功", roleName)}, nil
}

func handleListRoles(params map[string]interface{}) (interface{}, error) {
	sql := "SELECT ROLE FROM DBA_ROLES ORDER BY ROLE"
	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"roles": results, "count": len(results)}, nil
}

func handleListTablespaces(params map[string]interface{}) (interface{}, error) {
	sql := `SELECT
		t.TABLESPACE_NAME,
		t.STATUS,
		t.CONTENTS,
		d.FILE_NAME,
		d.BYTES / 1024 / 1024 AS SIZE_MB,
		d.AUTOEXTENSIBLE
		FROM DBA_TABLESPACES t
		JOIN DBA_DATA_FILES d ON t.TABLESPACE_NAME = d.TABLESPACE_NAME
		ORDER BY t.TABLESPACE_NAME`

	results, err := queryDB(sql)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"tablespaces": results, "count": len(results)}, nil
}

func handleCreateTablespace(params map[string]interface{}) (interface{}, error) {
	tsName := getString(params, "tablespace_name")
	if tsName == "" {
		return nil, fmt.Errorf("参数 tablespace_name 是必需的")
	}
	datafile := getString(params, "datafile")
	if datafile == "" {
		return nil, fmt.Errorf("参数 datafile 是必需的")
	}
	size := getInt(params, "size", 128)

	sql := fmt.Sprintf("CREATE TABLESPACE %s DATAFILE '%s' SIZE %d", tsName, datafile, size)
	if err := executeDDLDB(sql); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true, "message": fmt.Sprintf("表空间 %s 创建成功（%dMB）", tsName, size)}, nil
}
