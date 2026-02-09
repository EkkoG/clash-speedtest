package speedtester

import (
	"testing"
	"time"

	"github.com/metacubex/mihomo/adapter"
)

func TestTransferSummaryAdd(t *testing.T) {
	summary := newTransferSummary()
	summary.add(nil)

	errorMessage := "download request to https://example.com/__down?bytes=1 failed: boom"
	summary.add(&downloadResult{error: errorMessage})
	if summary.successCount != 0 {
		t.Fatalf("expected successCount to remain 0, got %d", summary.successCount)
	}
	if len(summary.errors) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(summary.errors))
	}
	if summary.errors[0] != errorMessage {
		t.Fatalf("expected error message %q, got %q", errorMessage, summary.errors[0])
	}

	summary.add(&downloadResult{error: errorMessage})
	if len(summary.errors) != 1 {
		t.Fatalf("expected duplicate errors to be deduplicated, got %d", len(summary.errors))
	}

	summary.add(&downloadResult{bytes: 100, duration: time.Second})
	summary.add(&downloadResult{bytes: 50, duration: 2 * time.Second})

	if summary.successCount != 2 {
		t.Fatalf("expected successCount to be 2, got %d", summary.successCount)
	}
	if summary.totalBytes != 150 {
		t.Fatalf("expected totalBytes to be 150, got %d", summary.totalBytes)
	}
	if summary.totalDuration != 3*time.Second {
		t.Fatalf("expected totalDuration to be 3s, got %v", summary.totalDuration)
	}
	if summary.averageDuration() != 1500*time.Millisecond {
		t.Fatalf("expected averageDuration to be 1.5s, got %v", summary.averageDuration())
	}
}

func TestResultFormatErrors(t *testing.T) {
	result := &Result{}
	if result.FormatDownloadError() != "N/A" {
		t.Fatalf("expected empty download error to format as N/A, got %q", result.FormatDownloadError())
	}
	if result.FormatUploadError() != "N/A" {
		t.Fatalf("expected empty upload error to format as N/A, got %q", result.FormatUploadError())
	}

	result.DownloadError = "download failed: timeout"
	result.UploadError = "upload failed: status 500"
	if result.FormatDownloadError() != result.DownloadError {
		t.Fatalf("expected download error to pass through, got %q", result.FormatDownloadError())
	}
	if result.FormatUploadError() != result.UploadError {
		t.Fatalf("expected upload error to pass through, got %q", result.FormatUploadError())
	}

	result.DownloadSpeed = 1024
	result.UploadSpeed = 2048
	if result.FormatDownloadSpeed() != result.DownloadError {
		t.Fatalf("expected download speed to prefer error string, got %q", result.FormatDownloadSpeed())
	}
	if result.FormatUploadSpeed() != result.UploadError {
		t.Fatalf("expected upload speed to prefer error string, got %q", result.FormatUploadSpeed())
	}
	if result.FormatDownloadSpeedValue() == result.DownloadError {
		t.Fatalf("expected download speed value to ignore error string")
	}
	if result.FormatUploadSpeedValue() == result.UploadError {
		t.Fatalf("expected upload speed value to ignore error string")
	}
}

func TestProxyDedupKey(t *testing.T) {
	mustParse := func(config map[string]any) *CProxy {
		proxy, err := adapter.ParseProxy(config)
		if err != nil {
			t.Fatalf("ParseProxy: %v", err)
		}
		return &CProxy{Proxy: proxy, Config: config}
	}

	// 相同 server+port+type 应得到相同 key
	p1 := mustParse(map[string]any{
		"name": "node1", "type": "vless", "server": "a.com", "port": 443,
		"uuid": "x", "network": "tcp",
	})
	p2 := mustParse(map[string]any{
		"name": "node2", "type": "vless", "server": "a.com", "port": 443,
		"uuid": "y", "network": "ws",
	})
	if k1, k2 := proxyDedupKey(p1), proxyDedupKey(p2); k1 != k2 {
		t.Errorf("same server+port+type should have same key: %q vs %q", k1, k2)
	}

	// port 为 float64（YAML 常见）
	p3 := mustParse(map[string]any{
		"name": "node3", "type": "vless", "server": "a.com", "port": float64(443),
		"uuid": "z", "network": "tcp",
	})
	if k1, k3 := proxyDedupKey(p1), proxyDedupKey(p3); k1 != k3 {
		t.Errorf("port int vs float64 should match: %q vs %q", k1, k3)
	}

	// 不同 server 或 port 或 type 应得到不同 key
	p4 := mustParse(map[string]any{
		"name": "node4", "type": "vless", "server": "b.com", "port": 443,
		"uuid": "x", "network": "tcp",
	})
	p5 := mustParse(map[string]any{
		"name": "node5", "type": "vless", "server": "a.com", "port": 8443,
		"uuid": "x", "network": "tcp",
	})
	p6 := mustParse(map[string]any{
		"name": "node6", "type": "vmess", "server": "a.com", "port": 443,
		"uuid": "x", "alterId": 0, "cipher": "auto",
	})
	keys := []string{proxyDedupKey(p1), proxyDedupKey(p4), proxyDedupKey(p5), proxyDedupKey(p6)}
	seen := make(map[string]bool)
	for _, k := range keys {
		if seen[k] {
			t.Errorf("expected unique keys for different server/port/type, duplicate: %q", k)
		}
		seen[k] = true
	}
}
