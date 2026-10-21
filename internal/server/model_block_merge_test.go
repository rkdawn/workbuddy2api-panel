package server

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/linguo2625469/workbuddy2api-panel/internal/auth"
)

// 显式分域请求不受另一域健康号影响；fork 宽限窗与积分保底在真实 HTTP
// 路径中保留状态码、原因与 Retry-After，不只验证单个判定函数。
func TestRealmModelBlockRetainsForkErrorPolicy(t *testing.T) {
	for _, tc := range []struct {
		name   string
		grace  time.Duration
		floor  bool
		status int
		code   string
		retry  bool
	}{
		{"grace enabled", 15 * time.Minute, false, http.StatusServiceUnavailable, "model_unavailable", true},
		{"grace disabled", 0, false, http.StatusBadRequest, "model_unavailable", false},
		{"combined floor", 15 * time.Minute, true, http.StatusBadRequest, "insufficient_credits", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			up := newFakeUpstream(t, func(authz string) (int, string, bool) {
				calls++
				return http.StatusOK, sseOK, true
			})
			p := newRealmPool(t)
			p.SetCrossDomainFallback(true)
			p.SetModelBlockGrace(tc.grace)
			p.CooldownSoftForModel("gl1", time.Minute, time.Now().Add(2*time.Minute), "shared-model", "6004 model rate limit")
			if tc.floor {
				p.SetCreditFloor(150)
				p.Add(&auth.Auth{UID: "poor", Domain: "www.workbuddy.ai", AccessToken: "at-poor", ExpiresAt: 9999999999})
				p.SetCredits("poor", 100, 0)
				p.NoteModelCost("poor", "shared-model", 2, 1000)
			}
			h := NewHandler(Config{Pool: p, Upstream: up})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest("POST", "/v1/chat/completions",
				strings.NewReader(`{"model":"global:shared-model","messages":[{"role":"user","content":"hi"}]}`)))
			if rec.Code != tc.status || !strings.Contains(rec.Body.String(), tc.code) {
				t.Fatalf("status=%d body=%s; want %d %s", rec.Code, rec.Body, tc.status, tc.code)
			}
			if calls != 0 {
				t.Fatalf("blocked explicit realm must not contact another account: calls=%d", calls)
			}
			if got := rec.Header().Get("Retry-After"); tc.retry {
				seconds, err := strconv.Atoi(got)
				if err != nil || seconds < 110 || seconds > 120 {
					t.Fatalf("Retry-After=%q; want remaining model cooldown", got)
				}
			} else if got != "" {
				t.Fatalf("non-retryable response has Retry-After=%q", got)
			}
			if tc.floor && !strings.Contains(rec.Body.String(), "1 below credit floor") {
				t.Fatalf("combined-state hint missing: %s", rec.Body)
			}
		})
	}
}
