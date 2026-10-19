package capture

import (
	"errors"
	"net/url"
	"testing"
)

func TestParseShareURL(t *testing.T) {
	deepLink := "yokasgs://?payload=1|24,example_20261015214518.sgs"
	shareURL := "https://www.sanguosha.cn/share/index.html?version=3.7.6&reqdata=" + url.QueryEscape(deepLink)

	result, err := ParseShareURL(shareURL)
	if err != nil {
		t.Fatalf("ParseShareURL() error = %v", err)
	}
	if result.DeepLink != deepLink {
		t.Fatalf("DeepLink = %q, want %q", result.DeepLink, deepLink)
	}
}

func TestParseShareURLRejectsUntrustedHost(t *testing.T) {
	deepLink := url.QueryEscape("yokasgs://?payload=example")
	_, err := ParseShareURL("https://example.com/share/index.html?reqdata=" + deepLink)
	if !errors.Is(err, ErrInvalidShareURL) {
		t.Fatalf("error = %v, want ErrInvalidShareURL", err)
	}
}

func TestParseShareURLRequiresGamePayload(t *testing.T) {
	shareURL := "https://www.sanguosha.cn/share/index.html?reqdata=" + url.QueryEscape("yokasgs://")
	if _, err := ParseShareURL(shareURL); err == nil {
		t.Fatal("ParseShareURL() error = nil, want validation error")
	}
}
