package config

import (
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	Listen            string
	Database          string
	Key               string
	FrontendDir       string
	PublicURL         string
	EpayGateway       string
	EpayPID           string
	EpayKey           string
	EpayType          string
	CodepayCreateURL  string
	CodepayMerchantID string
	CodepayKey        string
}

func Load() *Config {
	return &Config{
		Listen:            getEnv("LISTEN", "0.0.0.0:18888"),
		Database:          getEnv("DATABASE", "sqlite3://data.db"),
		Key:               getEnv("KEY", ""),
		FrontendDir:       getEnv("FRONTEND_DIR", "./public"),
		PublicURL:         getEnv("PUBLIC_URL", ""),
		EpayGateway:       getEnv("EPAY_GATEWAY", ""),
		EpayPID:           getEnv("EPAY_PID", ""),
		EpayKey:           getEnv("EPAY_KEY", ""),
		EpayType:          getEnv("EPAY_TYPE", "alipay"),
		CodepayCreateURL:  getEnv("CODEPAY_CREATE_URL", ""),
		CodepayMerchantID: getEnv("CODEPAY_MERCHANT_ID", ""),
		CodepayKey:        getEnv("CODEPAY_KEY", ""),
	}
}

func (c *Config) ValidatePayments() error {
	if c.PublicURL != "" {
		if err := validateAbsoluteURL(c.PublicURL, true); err != nil {
			return fmt.Errorf("PUBLIC_URL: %w", err)
		}
		u, _ := url.Parse(c.PublicURL)
		if u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("PUBLIC_URL: query and fragment are not allowed")
		}
	}
	epayAny := c.EpayGateway != "" || c.EpayPID != "" || c.EpayKey != ""
	if epayAny && (c.PublicURL == "" || c.EpayGateway == "" || c.EpayPID == "" || c.EpayKey == "" || c.EpayType == "") {
		return fmt.Errorf("EPAY configuration is incomplete")
	}
	codepayAny := c.CodepayCreateURL != "" || c.CodepayMerchantID != "" || c.CodepayKey != ""
	if codepayAny && (c.PublicURL == "" || c.CodepayCreateURL == "" || c.CodepayMerchantID == "" || c.CodepayKey == "") {
		return fmt.Errorf("CODEPAY configuration is incomplete")
	}
	for name, value := range map[string]string{"EPAY_GATEWAY": c.EpayGateway, "CODEPAY_CREATE_URL": c.CodepayCreateURL} {
		if value != "" {
			if err := validateAbsoluteURL(value, true); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	return nil
}

func validateAbsoluteURL(value string, httpsOnly bool) error {
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || (httpsOnly && u.Scheme != "https") {
		return fmt.Errorf("must be an absolute %s URL", map[bool]string{true: "HTTPS", false: "HTTP(S)"}[httpsOnly])
	}
	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
