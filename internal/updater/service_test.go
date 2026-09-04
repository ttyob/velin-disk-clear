package updater

import (
	"net/http"
	"net/url"
	"testing"
)

func TestNormaliseVersion(t *testing.T) {
	tests := map[string]string{"v0.2.0": "0.2.0", "1.4": "1.4.0", "  v10.20.3  ": "10.20.3", "bad": ""}
	for input, expected := range tests {
		if actual := normaliseVersion(input); actual != expected {
			t.Fatalf("normaliseVersion(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	if compareVersions("0.10.0", "0.9.9") <= 0 || compareVersions("0.2.0", "0.2.0") != 0 || compareVersions("0.1.9", "0.2.0") >= 0 {
		t.Fatal("version comparison is incorrect")
	}
}

func TestParseSHA256(t *testing.T) {
	digest, err := parseSHA256("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil || digest == "" {
		t.Fatalf("parseSHA256() error = %v", err)
	}
	if _, err := parseSHA256("sha256:not-a-digest"); err == nil {
		t.Fatal("expected invalid digest to be rejected")
	}
}

func TestValidateDownloadURL(t *testing.T) {
	if _, err := validateDownloadURL("https://github.com/ttyob/velin-disk-clear/releases/download/v0.2.0/Velin.Clear.exe"); err != nil {
		t.Fatalf("expected HTTPS URL to be accepted: %v", err)
	}
	if _, err := validateDownloadURL("https://updates.example.com/update.exe"); err != nil {
		t.Fatalf("expected HTTPS asset host to be accepted: %v", err)
	}
	if _, err := validateDownloadURL("http://updates.example.com/update.exe"); err == nil {
		t.Fatal("expected HTTP URL to be rejected")
	}
}

func TestSafeRedirectAllowsAnyHTTPSHost(t *testing.T) {
	assetURL, err := url.Parse("https://cdn.example.com/update.exe?sig=signed")
	if err != nil {
		t.Fatal(err)
	}
	if err := safeRedirect(&http.Request{URL: assetURL}, nil); err != nil {
		t.Fatalf("expected HTTPS redirect to be accepted: %v", err)
	}
	unsafeURL, err := url.Parse("http://cdn.example.com/update.exe")
	if err != nil {
		t.Fatal(err)
	}
	if err := safeRedirect(&http.Request{URL: unsafeURL}, nil); err == nil {
		t.Fatal("expected HTTP redirect to be rejected")
	}
}
