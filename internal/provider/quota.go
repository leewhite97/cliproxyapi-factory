package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"math"
	"net/http"
	"strings"
	"time"
)

type billingWindow struct {
	UsedPercent      *float64 `json:"usedPercent"`
	WindowEnd        *string  `json:"windowEnd"`
	SecondsRemaining *int64   `json:"secondsRemaining"`
}
type billingGroup struct {
	FiveHour *billingWindow `json:"fiveHour"`
	Weekly   *billingWindow `json:"weekly"`
	Monthly  *billingWindow `json:"monthly"`
}
type FactoryBillingLimitsResponse struct {
	Limits struct {
		Standard *billingGroup `json:"standard"`
		Core     *billingGroup `json:"core"`
	} `json:"limits"`
}

func ConvertBillingLimitsToQuota(raw []byte, now time.Time) (pluginapi.QuotaFetchResponse, error) {
	var resp FactoryBillingLimitsResponse
	if e := json.Unmarshal(raw, &resp); e != nil {
		return pluginapi.QuotaFetchResponse{}, fmt.Errorf("factory: invalid billing limits")
	}
	groups := []pluginapi.QuotaGroup{}
	for _, g := range []struct {
		name string
		data *billingGroup
		core bool
	}{{"Standard", resp.Limits.Standard, false}, {"Droid Core", resp.Limits.Core, true}} {
		if g.data == nil {
			continue
		}
		buckets := []pluginapi.QuotaBucket{}
		for _, w := range []struct {
			name string
			data *billingWindow
		}{{"5h", g.data.FiveHour}, {"weekly", g.data.Weekly}, {"monthly", g.data.Monthly}} {
			if w.data == nil || w.data.UsedPercent == nil {
				continue
			}
			// Factory may emit inactive Core placeholders with zero use and no window.
			if g.core && w.data.WindowEnd == nil && w.data.SecondsRemaining == nil {
				continue
			}
			used := *w.data.UsedPercent
			if math.IsNaN(used) || math.IsInf(used, 0) || used < 0 || used > 100 {
				return pluginapi.QuotaFetchResponse{}, fmt.Errorf("factory: invalid quota percentage")
			}
			reset := ""
			if w.data.WindowEnd != nil && strings.TrimSpace(*w.data.WindowEnd) != "" {
				reset = *w.data.WindowEnd
			} else if w.data.SecondsRemaining != nil && *w.data.SecondsRemaining > 0 {
				reset = now.Add(time.Duration(*w.data.SecondsRemaining) * time.Second).UTC().Format(time.RFC3339)
			}
			buckets = append(buckets, pluginapi.QuotaBucket{Window: w.name, RemainingFraction: (100 - used) / 100, ResetTime: reset, Description: fmt.Sprintf("%.0f%% remaining", 100-used)})
		}
		if len(buckets) > 0 {
			groups = append(groups, pluginapi.QuotaGroup{DisplayName: g.name, Buckets: buckets})
		}
	}
	if len(groups) == 0 {
		return pluginapi.QuotaFetchResponse{}, fmt.Errorf("factory: no observed quota windows")
	}
	// Billing overagePreference is not a subscription identifier.
	return pluginapi.QuotaFetchResponse{Groups: groups}, nil
}

func (s *Service) FetchQuota(ctx context.Context, callbackID string, storageJSON []byte) (pluginapi.QuotaFetchResponse, error) {
	c, err := credential(storageJSON)
	if err != nil {
		return pluginapi.QuotaFetchResponse{}, err
	}

	headers := make(http.Header)
	headers.Set("Authorization", "Bearer "+c.Key)
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Factory-Client", "cli")
	headers.Set("X-Client-Version", "0.162.1")
	headers.Set("User-Agent", "factory-cli/0.162.1")

	url := c.Endpoint + "/api/billing/limits"
	statusCode, _, body, err := s.Host.Do(ctx, "GET", callbackID, url, headers, nil)
	if err != nil {
		return pluginapi.QuotaFetchResponse{}, fmt.Errorf("factory: billing limits request failed: %w", err)
	}

	if statusCode != http.StatusOK {
		return pluginapi.QuotaFetchResponse{}, &StatusError{
			Code:    "upstream_quota_error",
			Message: fmt.Sprintf("factory: upstream billing limits returned HTTP %d: %s", statusCode, string(body)),
			Status:  statusCode,
		}
	}

	return ConvertBillingLimitsToQuota(body, time.Now())
}
