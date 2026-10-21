package pool

import (
	"testing"

	"github.com/linguo2625469/workbuddy2api-panel/internal/auth"
)

// 上游按域统计与 fork 积分保底组合：另一域的健康账号不掩盖本域阻塞，
// 无成本观测时仍使用本域目录倍率，而不是空域或另一域的价格。
func TestModelBlockedForRealmCombinedLockAndFloor(t *testing.T) {
	p := New("")
	p.SetCreditFloor(150)
	p.Add(&auth.Auth{UID: "cn", Domain: "www.codebuddy.cn"})
	p.Add(&auth.Auth{UID: "locked", Domain: "www.workbuddy.ai"})
	p.Add(&auth.Auth{UID: "poor", Domain: "www.workbuddy.ai"})
	p.SetCredits("cn", 1000, 0)
	p.SetCredits("poor", 100, 0)
	p.SetModelRateOf(func(realm, model string) string {
		if realm == "global" && model == "shared-model" {
			return "1.5"
		}
		return "0"
	})
	p.BlockModelBackoff("locked", "shared-model", "11158 model not allowed")

	st := p.ModelBlockedForRealm("shared-model", "global")
	if !st.Blocked || st.Count != 1 || st.FloorCount != 1 || st.Until.IsZero() {
		t.Fatalf("global lock + floor: %+v", st)
	}
	if st := p.ModelBlockedForRealm("shared-model", "cn"); st.Blocked {
		t.Fatalf("cn account is available: %+v", st)
	}
	if st := p.ModelBlocked("shared-model"); st.Blocked {
		t.Fatalf("all-realm query must still see healthy cn account: %+v", st)
	}

	p.SetModelRateOf(func(realm, model string) string { return "0" })
	if st := p.ModelBlockedForRealm("shared-model", "global"); st.Blocked {
		t.Fatalf("free global model must not be blocked by credit floor: %+v", st)
	}
}
