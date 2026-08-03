// Package signing provides Ed25519 signing and verification for runtime
// authorization packages. Canonical bytes follow RFC 8785 deterministic JSON:
// object keys are sorted lexicographically, integers remain integers, and the
// only excluded field is signature.
package signing

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// AuthorizationPackage is the signed runtime authorization sent to the WeKnora
// runtime for enforcement. Every field except Signature is included in the
// canonical bytes used for signing and verification.
type AuthorizationPackage struct {
	BillingCustomerID            string            `json:"billing_customer_id"`
	SubjectKeys                  []string          `json:"subject_keys"`
	PlanCode                     string            `json:"plan_code"`
	SubscriptionCode             string            `json:"subscription_code"`
	SubscriptionStatus           string            `json:"subscription_status"`
	EntitlementCodes             []string          `json:"entitlement_codes"`
	SpendableCredits             int64             `json:"spendable_credits"`
	EnterpriseAvailableCredits   int64             `json:"enterprise_available_credits"`
	AuthorizationCapacityCredits int64             `json:"authorization_capacity_credits"`
	CurrentPeriodStart           time.Time         `json:"current_period_start"`
	CurrentPeriodEnd             time.Time         `json:"current_period_end"`
	SnapshotVersion              int64             `json:"snapshot_version"`
	CoveredTenantSeq             int64             `json:"covered_tenant_seq"`
	RatePackageVersion           string            `json:"rate_package_version"`
	RatePackage                  []SignedRateEntry `json:"rate_package"`
	ExpiresAt                    time.Time         `json:"expires_at"`
	KeyID                        string            `json:"key_id"`
	Signature                    string            `json:"signature"`
}

// SignedRateEntry is a single customer-facing rate included in the signed
// package. Entries are sorted by (ResourceCode, Provider, Model) before signing.
type SignedRateEntry struct {
	ResourceCode   string `json:"resource_code"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	CreditsPerUnit int64  `json:"credits_per_unit"`
	UnitSize       int64  `json:"unit_size"`
}

// ErrInvalidSignature is returned when signature verification fails.
var ErrInvalidSignature = errors.New("signing: invalid signature")

// ErrNoMatchingKey is returned when no key matches the package's key_id during
// verification.
var ErrNoMatchingKey = errors.New("signing: no matching key for verification")

// CanonicalBytes returns the deterministic RFC 8785-style JSON bytes of the
// authorization package, excluding only the signature field. Subject keys,
// entitlement codes, and rate entries are sorted before serialization so that
// the same logical package always produces identical bytes.
func CanonicalBytes(pkg AuthorizationPackage) ([]byte, error) {
	// Zero out the signature so it does not influence the canonical form.
	pkg.Signature = ""

	// Sort slice fields for deterministic serialization.
	pkg.SubjectKeys = sortStringsCopy(pkg.SubjectKeys)
	pkg.EntitlementCodes = sortStringsCopy(pkg.EntitlementCodes)
	sortRateEntries(pkg.RatePackage)

	// First marshal: struct field order, valid JSON.
	raw, err := json.Marshal(pkg)
	if err != nil {
		return nil, fmt.Errorf("signing: marshal package: %w", err)
	}

	// Decode into generic tree with UseNumber so integers stay as json.Number
	// instead of degrading to float64.
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var tree map[string]any
	if err := dec.Decode(&tree); err != nil {
		return nil, fmt.Errorf("signing: decode package tree: %w", err)
	}

	// Remove the signature key from the canonical form.
	delete(tree, "signature")

	// Re-marshal: encoding/json sorts object keys lexicographically at every
	// depth, producing RFC 8785-compatible deterministic output.
	canonical, err := json.Marshal(tree)
	if err != nil {
		return nil, fmt.Errorf("signing: marshal canonical tree: %w", err)
	}

	return canonical, nil
}

// sortStringsCopy returns a sorted copy of s without mutating the original.
func sortStringsCopy(s []string) []string {
	if len(s) == 0 {
		return []string{}
	}
	out := make([]string, len(s))
	copy(out, s)
	sort.Strings(out)
	return out
}

// sortRateEntries sorts entries in place by (ResourceCode, Provider, Model).
func sortRateEntries(entries []SignedRateEntry) {
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.ResourceCode != b.ResourceCode {
			return a.ResourceCode < b.ResourceCode
		}
		if a.Provider != b.Provider {
			return a.Provider < b.Provider
		}
		return a.Model < b.Model
	})
}
