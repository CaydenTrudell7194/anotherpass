package handler

import "testing"

func TestNormalizeSiteSettingsClampsRuntimeValues(t *testing.T) {
	settings := SiteSettings{SiteName: " ", RegisterExpireDays: 0, ThemePolicy: "invalid", OfflineNodeSeconds: 1, OfflineNodeRetentionHours: 0}
	normalizeSiteSettings(&settings)
	if settings.SiteName != "转发面板" {
		t.Fatalf("unexpected site name: %q", settings.SiteName)
	}
	if settings.ThemePolicy != "classic" {
		t.Fatalf("unexpected theme: %q", settings.ThemePolicy)
	}
	if settings.RegisterExpireDays != 365 {
		t.Fatalf("unexpected expire days: %d", settings.RegisterExpireDays)
	}
	if settings.OfflineNodeSeconds != 20 {
		t.Fatalf("unexpected offline seconds: %d", settings.OfflineNodeSeconds)
	}
	if settings.OfflineNodeRetentionHours != 24 {
		t.Fatalf("unexpected retention: %d", settings.OfflineNodeRetentionHours)
	}
}

func TestDefaultSiteSettingsKeepsRegistrationClosed(t *testing.T) {
	settings := DefaultSiteSettings()
	if settings.AllowRegister {
		t.Fatal("registration must be disabled by default")
	}
	if settings.RegisterUserGroupID == 0 || settings.RegisterRuleLimit <= 0 {
		t.Fatal("registration defaults are incomplete")
	}
}
