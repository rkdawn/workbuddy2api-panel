// model_block_status_test.go handler.modelBlockStatus 梯度容错测试（fork 补丁）：
// 宽限窗关闭 = 上游原行为（一律 400）；开启时「解封临近 → 503 可重试，
// 解封尚远/未知/已过期 → 400」。纯判定函数级测试，不发起请求。
package server

import (
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/linguo2625469/workbuddy2api-panel/internal/auth"
	"github.com/linguo2625469/workbuddy2api-panel/internal/pool"
	"github.com/linguo2625469/workbuddy2api-panel/internal/upstream"
)

// TestSetRetryAfter Retry-After 头写入规则（fork 补丁）：
// 正值向上取整写秒；d<=0（已解封/无效）不写头——宁可缺省不给猜测值。
func TestSetRetryAfter(t *testing.T) {
	t.Run("positive ceil to seconds", func(t *testing.T) {
		rec := httptest.NewRecorder()
		setRetryAfter(rec, 1500*time.Millisecond)
		if got := rec.Header().Get("Retry-After"); got != "2" {
			t.Errorf("Retry-After=%q want 2 (1.5s ceil)", got)
		}
	})
	t.Run("exact second", func(t *testing.T) {
		rec := httptest.NewRecorder()
		setRetryAfter(rec, 90*time.Second)
		if got, _ := strconv.Atoi(rec.Header().Get("Retry-After")); got != 90 {
			t.Errorf("Retry-After=%q want 90", rec.Header().Get("Retry-After"))
		}
	})
	t.Run("nonpositive omits header", func(t *testing.T) {
		for _, d := range []time.Duration{0, -time.Second} {
			rec := httptest.NewRecorder()
			setRetryAfter(rec, d)
			if got := rec.Header().Get("Retry-After"); got != "" {
				t.Errorf("d=%v Retry-After=%q want omitted", d, got)
			}
		}
	})
}

// TestModelBlockStatusGradient 宽限窗四象限 + 关闭态逐字回归上游行为。
func TestModelBlockStatusGradient(t *testing.T) {
	p := testPoolWith(&auth.Auth{UID: "u1", AccessToken: "at", ExpiresAt: 9999999999})
	h := NewHandler(Config{Pool: p, Upstream: upstream.New()})

	mk := func(until time.Time) pool.ModelBlockStatus {
		return pool.ModelBlockStatus{Blocked: true, Count: 2, Until: until, Reason: "6004"}
	}

	t.Run("grace off keeps upstream 400", func(t *testing.T) {
		p.SetModelBlockGrace(0)
		if got := h.modelBlockStatus(mk(time.Now().Add(2 * time.Minute))); got != 400 {
			t.Errorf("grace=0 near-unblock: got %d want 400 (upstream behavior)", got)
		}
	})

	p.SetModelBlockGrace(15 * time.Minute)

	t.Run("near unblock returns 503", func(t *testing.T) {
		if got := h.modelBlockStatus(mk(time.Now().Add(5 * time.Minute))); got != 503 {
			t.Errorf("until=+5m within 15m grace: got %d want 503", got)
		}
	})

	t.Run("far unblock returns 400", func(t *testing.T) {
		if got := h.modelBlockStatus(mk(time.Now().Add(time.Hour))); got != 400 {
			t.Errorf("until=+1h beyond 15m grace: got %d want 400", got)
		}
	})

	t.Run("zero until returns 400", func(t *testing.T) {
		if got := h.modelBlockStatus(mk(time.Time{})); got != 400 {
			t.Errorf("until zero: got %d want 400 (unknown reset)", got)
		}
	})

	t.Run("expired until returns 400", func(t *testing.T) {
		if got := h.modelBlockStatus(mk(time.Now().Add(-time.Minute))); got != 400 {
			t.Errorf("until past: got %d want 400 (next pick recovers)", got)
		}
	})

	t.Run("grace window edge inclusive", func(t *testing.T) {
		// 边界含等号：until 恰在窗内（距边界还差 10s，规避时钟抖动）。
		if got := h.modelBlockStatus(mk(time.Now().Add(15*time.Minute - 10*time.Second))); got != 503 {
			t.Errorf("until just inside grace: got %d want 503", got)
		}
	})
}
