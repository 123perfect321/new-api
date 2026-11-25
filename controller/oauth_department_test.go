package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/oauth"
)

func TestOAuthDepartmentGroupPrefersOrgName(t *testing.T) {
	user := &oauth.OAuthUser{Extra: map[string]any{
		"org_id":   "org-123",
		"org_name": "研发中心",
	}}
	if got := oauthDepartmentGroup(user); got != "研发中心" {
		t.Fatalf("expected org_name as group, got %q", got)
	}
}

func TestOAuthDepartmentGroupFallsBackToOrgID(t *testing.T) {
	user := &oauth.OAuthUser{Extra: map[string]any{
		"org_id": "org-123",
	}}
	if got := oauthDepartmentGroup(user); got != "org-123" {
		t.Fatalf("expected org_id as group, got %q", got)
	}
}

func TestOAuthDepartmentGroupSanitizesName(t *testing.T) {
	user := &oauth.OAuthUser{Extra: map[string]any{
		"org_name": "  研发中心,  平台组  ",
	}}
	if got := oauthDepartmentGroup(user); got != "研发中心- 平台组" {
		t.Fatalf("unexpected sanitized group: %q", got)
	}
}

func TestOAuthDepartmentInitialQuotaUsesConfiguredDepartment(t *testing.T) {
	oldUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldUnit })

	t.Setenv(oauthDepartmentQuotasEnv, `{"org-123": 1200}`)
	t.Setenv(oauthDefaultQuotaCreditsEnv, "800")

	got := oauthDepartmentInitialQuota("org-123", "研发中心", "研发中心")
	if want := 600000000; got != want {
		t.Fatalf("expected %d, got %d", want, got)
	}
}

func TestOAuthDepartmentInitialQuotaFallsBackToDefault(t *testing.T) {
	oldUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldUnit })

	t.Setenv(oauthDepartmentQuotasEnv, `{}`)
	t.Setenv(oauthDefaultQuotaCreditsEnv, "300")

	got := oauthDepartmentInitialQuota("org-unknown", "未知部门", "未知部门")
	if want := 150000000; got != want {
		t.Fatalf("expected %d, got %d", want, got)
	}
}

func TestOAuthDepartmentInitialQuotaHandlesInvalidJSON(t *testing.T) {
	oldUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldUnit })

	t.Setenv(oauthDepartmentQuotasEnv, "{invalid")
	t.Setenv(oauthDefaultQuotaCreditsEnv, "800")

	got := oauthDepartmentInitialQuota("org-unknown", "未知部门", "未知部门")
	if want := 400000000; got != want {
		t.Fatalf("expected %d, got %d", want, got)
	}
}
