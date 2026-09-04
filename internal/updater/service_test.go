package updater

import "testing"

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
		t.Fatalf("expected GitHub URL to be accepted: %v", err)
	}
	if _, err := validateDownloadURL("https://example.com/update.exe"); err == nil {
		t.Fatal("expected non-GitHub URL to be rejected")
	}
}
