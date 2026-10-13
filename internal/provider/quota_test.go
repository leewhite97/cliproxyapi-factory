package provider

import (
	"encoding/json"
	"testing"
	"time"
)

func TestConvertBillingLimitsToQuota(t *testing.T) {
	fixture := `{
  "usesTokenRateLimitsBilling": true,
  "limits": {
    "standard": {
      "fiveHour": {
        "usedPercent": 6,
        "windowEnd": "2026-10-08T12:37:11.880Z",
        "secondsRemaining": 9883
      },
      "weekly": {
        "usedPercent": 5,
        "windowEnd": "2026-10-08T20:20:07.620Z",
        "secondsRemaining": 37659
      },
      "monthly": {
        "usedPercent": 3,
        "windowEnd": "2026-10-31T20:20:07.620Z",
        "secondsRemaining": 2024859
      }
    },
    "core": {
      "fiveHour": {
        "usedPercent": 0,
        "windowEnd": null,
        "secondsRemaining": null
      },
      "weekly": {
        "usedPercent": 0,
        "windowEnd": null,
        "secondsRemaining": null
      },
      "monthly": {
        "usedPercent": 0,
        "windowEnd": null,
        "secondsRemaining": null
      }
    }
  },
  "overagePreference": "droidCore",
  "canManageOverage": true,
  "extraUsageBalanceCents": 0,
  "extraUsageAllowed": true
}`

	now := time.Now()
	res, err := ConvertBillingLimitsToQuota([]byte(fixture), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Subscription != nil {
		t.Errorf("unexpected inferred plan, got %+v", res.Subscription)
	}

	if len(res.Groups) != 1 {
		t.Fatalf("expected 1 group (Standard only, Core has null windows), got %d", len(res.Groups))
	}

	std := res.Groups[0]
	if std.DisplayName != "Standard" || len(std.Buckets) != 3 {
		t.Fatalf("unexpected standard group: %+v", std)
	}

	b5h := std.Buckets[0]
	if b5h.Window != "5h" || b5h.RemainingFraction < 0.939 || b5h.RemainingFraction > 0.941 {
		t.Errorf("unexpected 5h bucket: %+v", b5h)
	}
	if b5h.ResetTime != "2026-10-08T12:37:11.880Z" {
		t.Errorf("unexpected reset time: %s", b5h.ResetTime)
	}

	bw := std.Buckets[1]
	if bw.Window != "weekly" || bw.RemainingFraction < 0.949 || bw.RemainingFraction > 0.951 {
		t.Errorf("unexpected weekly bucket: %+v", bw)
	}

	bm := std.Buckets[2]
	if bm.Window != "monthly" || bm.RemainingFraction < 0.969 || bm.RemainingFraction > 0.971 {
		t.Errorf("unexpected monthly bucket: %+v", bm)
	}

	rawJSON, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}
	t.Logf("Normalized JSON: %s", string(rawJSON))
}
