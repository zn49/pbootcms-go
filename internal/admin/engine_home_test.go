package admin

import (
	"strings"
	"testing"
)

func TestHomeArrowVarsAndNestedIf(t *testing.T) {
	d := Data{
		Vars: map[string]any{
			"dbsecurity": "1",
			"user_info":  map[string]string{"username": "admin", "realname": "管", "login_count": "3", "update_time": "t", "last_login_ip": "2130706433"},
			"server":     map[string]string{"php_os": "windows", "server_name": "localhost", "server_addr": "127.0.0.1", "server_port": "8082", "php_version": "go", "php_version_low": "", "db_driver": "sqlite", "server_software": "gboot", "upload_max_filesize": "20M", "post_max_size": "20M"},
			"sum_msg":    "0",
			"model_msg":  []map[string]string{},
		},
		Session: map[string]string{"pwsecurity": "1", "ucode": "10001"},
		Funs: map[string]func([]string) string{
			"long2ip_safe": func(args []string) string {
				if len(args) == 0 {
					return ""
				}
				return long2ipSafe(args[0])
			},
		},
	}
	s := `{if(![$dbsecurity]||![$session.pwsecurity])}
bad
{if(![$dbsecurity])}db{/if}
{if(![$session.pwsecurity])}pw{/if}
{/if}
当前登录用户：{$user_info->username} {$user_info->realname}，登录IP：{fun=long2ip_safe([$user_info->last_login_ip])}
主机：{$server->php_os} {$server->server_name}`
	out := renderBody(s, d)
	if strings.Contains(out, "bad") || strings.Contains(out, "{/if}") || strings.Contains(out, "{\\") {
		t.Fatalf("nested if leftover: %s", out)
	}
	if !strings.Contains(out, "admin") || !strings.Contains(out, "127.0.0.1") || !strings.Contains(out, "windows") {
		t.Fatalf("vars: %s", out)
	}
	if strings.Contains(out, "{$") || strings.Contains(out, "{fun=") {
		t.Fatalf("unexpanded: %s", out)
	}
}

func TestLong2ipSafe(t *testing.T) {
	if got := long2ipSafe("2130706433"); got != "127.0.0.1" {
		t.Fatalf("%s", got)
	}
	if got := long2ipSafe("10.0.0.1"); got != "10.0.0.1" {
		t.Fatalf("%s", got)
	}
}
