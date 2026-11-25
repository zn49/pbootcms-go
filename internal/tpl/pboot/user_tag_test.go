package pboot

import (
	"strings"
	"testing"
)

func TestUserTags(t *testing.T) {
	ctx := &Ctx{Member: map[string]string{
		"username": "u1", "nickname": "昵", "register_time": "2020-01-01",
		"login_count": "3", "password": "secret",
	}}
	out := replaceTokens(`{user:username}|{user:nickname}|{user:registertime}|{user:logincount}|{user:password}`, ctx)
	if !strings.Contains(out, "u1") || !strings.Contains(out, "昵") || !strings.Contains(out, "2020-01-01") || !strings.Contains(out, "3") {
		t.Fatalf("%s", out)
	}
	if strings.Contains(out, "secret") {
		t.Fatalf("password leaked: %s", out)
	}
}

func TestAPITags(t *testing.T) {
	ctx := &Ctx{ClientIP: "1.2.3.4", apiStamp: "100"}
	// DB nil → empty appid/secret still forms signature of ""+""+"100"
	out := replaceTokens(`{pboot:timestamp}|{pboot:signature}|{pboot:userip}|{pboot:appid}`, ctx)
	if !strings.Contains(out, "100") || !strings.Contains(out, "1.2.3.4") {
		t.Fatalf("%s", out)
	}
	if strings.Contains(out, "{pboot:") {
		t.Fatalf("leftover: %s", out)
	}
	sig := ctx.apiSignature()
	if !strings.Contains(out, sig) {
		t.Fatalf("sig missing: %s want %s", out, sig)
	}
}
