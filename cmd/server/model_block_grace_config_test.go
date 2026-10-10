// model_block_grace_config_test.go pool.model_block_grace 配置测试（fork 补丁）：
// 默认关闭（0，上游原行为零回归）/ 文件解析 / 空值关闭 / 负值钳 0 / 非法时长报错。
package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestModelBlockGraceDefault 键缺席 → 默认 0（宽限窗关闭，末端行为与上游逐字一致）。
func TestModelBlockGraceDefault(t *testing.T) {
	c := Default()
	if err := c.normalize(); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if c.ModelBlockGraceDur != 0 {
		t.Errorf("model_block_grace=%v want 0 (default off)", c.ModelBlockGraceDur)
	}
}

// TestModelBlockGraceParsedFromFile 显式配置生效（如 "15m"）。
func TestModelBlockGraceParsedFromFile(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "c.json")
	os.WriteFile(fp, []byte(`{"pool":{"model_block_grace":"15m"}}`), 0o600)
	c, err := Load(fp)
	if err != nil {
		t.Fatal(err)
	}
	if c.ModelBlockGraceDur != 15*time.Minute {
		t.Errorf("model_block_grace=%v want 15m", c.ModelBlockGraceDur)
	}
}

// TestModelBlockGraceEmptyOff 空字符串视作关闭（0），不报 ParseDuration 错。
func TestModelBlockGraceEmptyOff(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "c.json")
	os.WriteFile(fp, []byte(`{"pool":{"model_block_grace":""}}`), 0o600)
	c, err := Load(fp)
	if err != nil {
		t.Fatal(err)
	}
	if c.ModelBlockGraceDur != 0 {
		t.Errorf("model_block_grace=%v want 0 (empty off)", c.ModelBlockGraceDur)
	}
}

// TestModelBlockGraceNegativeClamped 负值钳 0（非法即关闭，不报错：误写不炸启动）。
func TestModelBlockGraceNegativeClamped(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "c.json")
	os.WriteFile(fp, []byte(`{"pool":{"model_block_grace":"-5m"}}`), 0o600)
	c, err := Load(fp)
	if err != nil {
		t.Fatal(err)
	}
	if c.ModelBlockGraceDur != 0 {
		t.Errorf("model_block_grace=%v want 0 (negative clamped)", c.ModelBlockGraceDur)
	}
}

// TestModelBlockGraceInvalidErrors 非法时长启动报错（fail fast，风格同 breaker_cooldown）。
func TestModelBlockGraceInvalidErrors(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "c.json")
	os.WriteFile(fp, []byte(`{"pool":{"model_block_grace":"oops"}}`), 0o600)
	if _, err := Load(fp); err == nil {
		t.Fatal("want error for bad model_block_grace")
	}
}
