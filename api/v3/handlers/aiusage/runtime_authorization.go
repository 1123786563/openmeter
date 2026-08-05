package aiusage

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/aiusage/signing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

// authorizationPackageView is the normalized authorization package that the
// handler maps to the API RuntimeAuthorization response.
type authorizationPackageView struct {
	ContractVersion           string
	Authorized                bool
	AvailableCredits          int64
	ReservationCeilingCredits int64
	CoveredTenantSeq          int64
	DenialReason              string
}

type (
	GetCustomerRuntimeAuthorizationParams struct {
		CustomerID string
		Filter     *api.AIUsageRuntimeAuthorizationQuery
	}

	GetCustomerRuntimeAuthorizationRequest struct {
		Namespace   string
		CustomerID  string
		SubjectKeys []string
	}

	GetCustomerRuntimeAuthorizationResponse = api.AIUsageRuntimeAuthorization

	GetCustomerRuntimeAuthorizationHandler = httptransport.HandlerWithArgs[GetCustomerRuntimeAuthorizationRequest, GetCustomerRuntimeAuthorizationResponse, GetCustomerRuntimeAuthorizationParams]
)

// contractVersion is the frozen Phase 1 contract version string returned in
// every runtime authorization response.
const contractVersion = "weknora-billing-p1-v1"

// GetCustomerRuntimeAuthorization implements GET
// /customers/{customerId}/runtime-authorization. When the runtime
// authorization service is nil the operation returns 501.
func (h *handler) GetCustomerRuntimeAuthorization() GetCustomerRuntimeAuthorizationHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCustomerRuntimeAuthorizationParams) (GetCustomerRuntimeAuthorizationRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerRuntimeAuthorizationRequest{}, err
			}

			var subjectKeys []string
			if params.Filter != nil && params.Filter.SubjectKey != nil {
				subjectKeys = append(subjectKeys, *params.Filter.SubjectKey)
			}

			return GetCustomerRuntimeAuthorizationRequest{
				Namespace:   ns,
				CustomerID:  params.CustomerID,
				SubjectKeys: subjectKeys,
			}, nil
		},
		func(ctx context.Context, req GetCustomerRuntimeAuthorizationRequest) (GetCustomerRuntimeAuthorizationResponse, error) {
			if h.runtimeAuthorizationService == nil {
				return GetCustomerRuntimeAuthorizationResponse{}, models.NewGenericNotImplementedError(nil)
			}

			pkg, err := h.runtimeAuthorizationService.Get(ctx, req.CustomerID, req.SubjectKeys)
			if err != nil {
				return GetCustomerRuntimeAuthorizationResponse{}, err
			}

			return signedPackageToAPI(pkg, time.Now().UTC()), nil
		},
		commonhttp.JSONResponseEncoder[GetCustomerRuntimeAuthorizationResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-customer-runtime-authorization"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// authorizationPackageFromSigned maps the signed authorization package to the
// handler view. The customer is authorized when authorization_capacity_credits
// is positive.
func authorizationPackageFromSigned(pkg signing.AuthorizationPackage) authorizationPackageView {
	return authorizationPackageView{
		ContractVersion:           contractVersion,
		Authorized:                pkg.AuthorizationCapacityCredits > 0,
		AvailableCredits:          pkg.SpendableCredits,
		ReservationCeilingCredits: 0,
		CoveredTenantSeq:          pkg.CoveredTenantSeq,
	}
}

// signedPackageToAPI maps the signed authorization package to the API
// response, preserving the Ed25519 signature envelope so consumers (WeKnora)
// can cryptographically verify the authorization decision. The canonical
// bytes are recomputed (idempotent — CanonicalBytes zeroes the signature),
// SHA-256 hashed, and the hex signature is converted to base64 to match the
// WeKnora verification contract.
func signedPackageToAPI(pkg signing.AuthorizationPackage, now time.Time) api.AIUsageRuntimeAuthorization {
	resp := api.AIUsageRuntimeAuthorization{
		ContractVersion:  contractVersion,
		RetrievedAt:      now,
		Authorized:       pkg.AuthorizationCapacityCredits > 0,
		AvailableCredits: pkg.SpendableCredits,
		CoveredTenantSeq: pkg.CoveredTenantSeq,
		SnapshotVersion:  pkg.SnapshotVersion,
		KeyID:            pkg.KeyID,
	}

	if len(pkg.SubjectKeys) > 0 {
		resp.SubjectKey = pkg.SubjectKeys[0]
	}

	if !pkg.ExpiresAt.IsZero() {
		expires := pkg.ExpiresAt
		resp.ValidUntil = &expires
	}

	if pkg.AuthorizationCapacityCredits <= 0 {
		reason := "insufficient_credits"
		resp.DenialReason = &reason
	}

	// Recompute canonical bytes (idempotent) and derive the signing envelope.
	canonical, err := signing.CanonicalBytes(pkg)
	if err != nil {
		return resp // canonicalization failed; return unsigned
	}
	resp.CanonicalPayload = json.RawMessage(canonical)

	hash := sha256.Sum256(canonical)
	resp.CanonicalSHA256 = hex.EncodeToString(hash[:])

	if sigBytes, err := hex.DecodeString(pkg.Signature); err == nil {
		resp.Signature = base64.StdEncoding.EncodeToString(sigBytes)
	}

	return resp
}
