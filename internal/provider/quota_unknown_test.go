package provider

import (
	"testing"
	"time"
)

func TestQuotaMissingValuesAreNotFull(t *testing.T) {
	for _, raw := range []string{`{}`, `null`, `{"limits":{"standard":{"fiveHour":{}}}}`, `{"limits":{"standard":{"fiveHour":{"usedPercent":null}}}}`} {
		r, e := ConvertBillingLimitsToQuota([]byte(raw), time.Unix(0, 0))
		if e == nil && len(r.Groups) > 0 {
			t.Fatalf("missing quota became known: %s %+v", raw, r)
		}
	}
	raw := `{"limits":{"standard":{"fiveHour":{"usedPercent":0}}}}`
	r, e := ConvertBillingLimitsToQuota([]byte(raw), time.Unix(0, 0))
	if e != nil || len(r.Groups) != 1 || len(r.Groups[0].Buckets) != 1 || r.Groups[0].Buckets[0].RemainingFraction != 1 {
		t.Fatalf("real zero lost: %v %+v", e, r)
	}
}
func TestOveragePreferenceIsNotSubscriptionPlan(t *testing.T) {
	r, e := ConvertBillingLimitsToQuota([]byte(`{"overagePreference":"droidCore","limits":{"standard":{"fiveHour":{"usedPercent":1}}}}`), time.Unix(0, 0))
	if e != nil {
		t.Fatal(e)
	}
	if r.Subscription != nil && r.Subscription.Plan == "droidCore" {
		t.Fatal("overage preference passed as subscription")
	}
}
