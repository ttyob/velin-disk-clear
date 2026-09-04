package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ai-clear/internal/network"
)

const (
	latestReleaseURL = "https://api.github.com/repos/ttyob/velin-disk-clear/releases/latest"
	maxUpdateSize    = 512 << 20
)

type Info struct {
	Available      bool
	CurrentVersion string
	LatestVersion  string
	TagName        string
	ReleaseName    string
	Notes          string
	PublishedAt    string
	AssetName      string
	AssetSize      int64
	AssetURL       string
	Digest         string
	CheckedAt      string
}

type Downloaded struct {
	Version  string
	Size     int64
	Verified bool
}

type Service struct {
	dataDir string
	client  *http.Client
	mu      sync.Mutex
	path    string
	version string
	proxy   string
}

type release struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	PublishedAt string  `json:"published_at"`
	Assets      []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

func New(dataDir string) *Service {
	return &Service{
		dataDir: dataDir,
		client:  &http.Client{Timeout: 30 * time.Second, CheckRedirect: safeRedirect},
	}
}

func (s *Service) SetProxy(proxyURL string) error {
	client, err := network.HTTPClient(proxyURL, 30*time.Second, safeRedirect)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.proxy = strings.TrimSpace(proxyURL)
	s.client = client
	s.mu.Unlock()
	return nil
}

func (s *Service) Proxy() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.proxy
}

func (s *Service) httpClient() *http.Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client
}

func (s *Service) Check(ctx context.Context, currentVersion string) (Info, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	requestCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return Info{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "Velin-Clear-Updater/1")
	response, err := s.httpClient().Do(request)
	if err != nil {
		return Info{}, fmt.Errorf("check latest release: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Info{}, fmt.Errorf("check latest release: HTTP %d", response.StatusCode)
	}
	var latest release
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&latest); err != nil {
		return Info{}, fmt.Errorf("decode latest release: %w", err)
	}
	version := normaliseVersion(latest.TagName)
	if version == "" {
		return Info{}, errors.New("latest release has no valid version tag")
	}
	selected, ok := selectWindowsAsset(latest.Assets)
	if !ok {
		return Info{}, errors.New("latest release has no Velin Clear Windows asset")
	}
	if selected.Size <= 0 || selected.Size > maxUpdateSize {
		return Info{}, fmt.Errorf("latest release asset has invalid size %d", selected.Size)
	}
	if _, err := parseSHA256(selected.Digest); err != nil {
		return Info{}, fmt.Errorf("latest release asset digest is invalid: %w", err)
	}
	return Info{
		Available:      compareVersions(version, normaliseVersion(currentVersion)) > 0,
		CurrentVersion: normaliseVersion(currentVersion),
		LatestVersion:  version,
		TagName:        latest.TagName,
		ReleaseName:    latest.Name,
		Notes:          latest.Body,
		PublishedAt:    latest.PublishedAt,
		AssetName:      selected.Name,
		AssetSize:      selected.Size,
		AssetURL:       selected.BrowserDownloadURL,
		Digest:         selected.Digest,
		CheckedAt:      time.Now().Format(time.RFC3339),
	}, nil
}

func (s *Service) Download(ctx context.Context, currentVersion string) (Downloaded, error) {
	info, err := s.Check(ctx, currentVersion)
	if err != nil {
		return Downloaded{}, err
	}
	if !info.Available {
		return Downloaded{}, errors.New("already using the latest version")
	}
	downloadURL, err := validateDownloadURL(info.AssetURL)
	if err != nil {
		return Downloaded{}, err
	}
	if err := os.MkdirAll(filepath.Join(s.dataDir, "updates"), 0o700); err != nil {
		return Downloaded{}, fmt.Errorf("create update directory: %w", err)
	}
	updateDir := filepath.Join(s.dataDir, "updates")
	temporary, err := os.CreateTemp(updateDir, ".Velin.Clear-*.download")
	if err != nil {
		return Downloaded{}, fmt.Errorf("create update download: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()

	requestCtx, cancel := context.WithTimeout(ctxOrBackground(ctx), 10*time.Minute)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return Downloaded{}, err
	}
	request.Header.Set("Accept", "application/octet-stream")
	request.Header.Set("User-Agent", "Velin-Clear-Updater/1")
	response, err := s.httpClient().Do(request)
	if err != nil {
		return Downloaded{}, fmt.Errorf("download update: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Downloaded{}, fmt.Errorf("download update: HTTP %d", response.StatusCode)
	}
	if response.ContentLength > maxUpdateSize {
		return Downloaded{}, errors.New("update package is too large")
	}
	hash := sha256.New()
	writer := io.MultiWriter(temporary, hash)
	size, err := io.Copy(writer, io.LimitReader(response.Body, maxUpdateSize+1))
	if err != nil {
		return Downloaded{}, fmt.Errorf("save update package: %w", err)
	}
	if size <= 0 || size > maxUpdateSize {
		return Downloaded{}, errors.New("downloaded update package has invalid size")
	}
	if err := temporary.Close(); err != nil {
		return Downloaded{}, err
	}
	expected, _ := parseSHA256(info.Digest)
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return Downloaded{}, fmt.Errorf("update checksum mismatch: got %s", actual)
	}
	finalPath := filepath.Join(updateDir, "Velin.Clear-"+safeFilePart(info.LatestVersion)+".exe")
	if err := os.Rename(temporaryPath, finalPath); err != nil {
		return Downloaded{}, fmt.Errorf("store update package: %w", err)
	}
	s.mu.Lock()
	s.path, s.version = finalPath, info.LatestVersion
	s.mu.Unlock()
	return Downloaded{Version: info.LatestVersion, Size: size, Verified: true}, nil
}

func (s *Service) Install() error {
	s.mu.Lock()
	path, version := s.path, s.version
	s.mu.Unlock()
	if path == "" || version == "" {
		return errors.New("download an update before installing")
	}
	if _, err := os.Stat(path); err != nil {
		return errors.New("downloaded update package is no longer available")
	}
	target, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate current executable: %w", err)
	}
	return installUpdate(path, target, os.Getpid())
}

func selectWindowsAsset(assets []asset) (asset, bool) {
	for _, candidate := range assets {
		if strings.EqualFold(candidate.Name, "Velin.Clear.exe") {
			return candidate, true
		}
	}
	return asset{}, false
}

func validateDownloadURL(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", errors.New("update download URL must use HTTPS")
	}
	return parsed.String(), nil
}

func safeRedirect(request *http.Request, via []*http.Request) error {
	if request.URL.Scheme != "https" || request.URL.Host == "" {
		return errors.New("update redirect is not HTTPS")
	}
	return nil
}

func normaliseVersion(value string) string {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	versionParts := strings.FieldsFunc(value, func(r rune) bool { return r == '-' || r == '+' })
	if len(versionParts) == 0 {
		return ""
	}
	value = versionParts[0]
	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return ""
	}
	for index := 0; index < 3 && index < len(parts); index++ {
		if parts[index] == "" {
			return ""
		}
		for _, char := range parts[index] {
			if char < '0' || char > '9' {
				return ""
			}
		}
	}
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	return strings.Join(parts[:3], ".")
}

func compareVersions(left, right string) int {
	for index, leftPart := range strings.Split(normaliseVersion(left), ".") {
		leftValue := atoi(leftPart)
		rightParts := strings.Split(normaliseVersion(right), ".")
		rightValue := 0
		if index < len(rightParts) {
			rightValue = atoi(rightParts[index])
		}
		if leftValue != rightValue {
			if leftValue > rightValue {
				return 1
			}
			return -1
		}
	}
	return 0
}

func atoi(value string) int {
	result := 0
	for _, char := range value {
		result = result*10 + int(char-'0')
	}
	return result
}

func parseSHA256(value string) (string, error) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "sha256:"))
	if len(value) != sha256.Size*2 {
		return "", errors.New("SHA-256 digest must contain 64 hexadecimal characters")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", errors.New("SHA-256 digest is not hexadecimal")
	}
	return strings.ToLower(value), nil
}

func safeFilePart(value string) string {
	value = strings.Map(func(char rune) rune {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '.' || char == '-' {
			return char
		}
		return '_'
	}, value)
	return strings.Trim(value, ".-")
}

func ctxOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
