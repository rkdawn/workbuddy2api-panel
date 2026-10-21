package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 同一 pool 配置同时保留个人宽限窗与上游跨域开关，序列化后不丢键。
func TestSyncConfigKeepsGraceAndCrossDomainFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"pool":{"model_block_grace":"15m","cross_domain_fallback":true}}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ModelBlockGraceDur != 15*time.Minute || !cfg.Pool.CrossDomainFallback {
		t.Fatalf("grace=%v fallback=%v", cfg.ModelBlockGraceDur, cfg.Pool.CrossDomainFallback)
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	restored, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if restored.ModelBlockGraceDur != cfg.ModelBlockGraceDur || !restored.Pool.CrossDomainFallback {
		t.Fatalf("round trip: grace=%v fallback=%v", restored.ModelBlockGraceDur, restored.Pool.CrossDomainFallback)
	}
}
