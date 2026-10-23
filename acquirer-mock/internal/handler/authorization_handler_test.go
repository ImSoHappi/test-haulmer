package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ImSoHappi/test-haulmer/acquirer-mock/internal/domain"
	"github.com/ImSoHappi/test-haulmer/acquirer-mock/internal/handler"
	"github.com/ImSoHappi/test-haulmer/acquirer-mock/internal/storage"
)

func setupTestServer(t *testing.T) (*httptest.Server, *storage.MemoryStore) {
	t.Helper()
	store := storage.NewMemoryStore()
	h := handler.NewAuthorizationHandler(store, handler.WithTimeoutDelay(10*time.Millisecond))

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	return ts, store
}

func buildRequestPayload(pan, refID string, amount int64) domain.AuthorizationRequest {
	return domain.AuthorizationRequest{
		ReferenceID: refID,
		Amount:      amount,
		Currency:    "CLP",
		Card: domain.CardPayload{
			Number:      pan,
			ExpiryMonth: 12,
			ExpiryYear:  2028,
			CVV:         "123",
			HolderName:  "Test Cardholder",
		},
	}
}

func TestAuthorize_ApprovedCard(t *testing.T) {
	ts, _ := setupTestServer(t)
	payload := buildRequestPayload(domain.PANApproved, "ref-approved-001", 25990)

	bodyBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/authorizations", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-test-001")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "corr-test-001", resp.Header.Get("X-Correlation-ID"))

	var authResp domain.AuthorizationResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	require.NoError(t, err)

	assert.Equal(t, "ref-approved-001", authResp.ReferenceID)
	assert.Equal(t, "APPROVED", authResp.Status)
	assert.Equal(t, "00", authResp.Code)
	assert.NotEmpty(t, authResp.AuthorizationCode)
	assert.Contains(t, authResp.AuthorizationCode, "AUT-")
	assert.Equal(t, int64(25990), authResp.Amount)
	assert.Equal(t, "CLP", authResp.Currency)

	// Verify subsequent GET returns the approved record
	getReq, err := http.NewRequest(http.MethodGet, ts.URL+"/authorizations/ref-approved-001", nil)
	require.NoError(t, err)
	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()

	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	var getAuthResp domain.AuthorizationResponse
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&getAuthResp))
	assert.Equal(t, authResp.AuthorizationCode, getAuthResp.AuthorizationCode)
}

func TestAuthorize_DeclinedCard_InsufficientFunds(t *testing.T) {
	ts, _ := setupTestServer(t)
	payload := buildRequestPayload(domain.PANInsufficientFunds, "ref-declined-002", 99990)

	bodyBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/authorizations", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var authResp domain.AuthorizationResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	require.NoError(t, err)

	assert.Equal(t, "ref-declined-002", authResp.ReferenceID)
	assert.Equal(t, "DECLINED", authResp.Status)
	assert.Equal(t, "51", authResp.Code)
	assert.Equal(t, "insufficient_funds", authResp.DeclineCode)
	assert.Contains(t, authResp.Message, "insufficient funds")
}

func TestAuthorize_Persistent503(t *testing.T) {
	ts, _ := setupTestServer(t)
	payload := buildRequestPayload(domain.PANAcquirerDown, "ref-503-003", 10000)

	bodyBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/authorizations", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	var errResp domain.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Equal(t, "service_unavailable", errResp.Error)
}

func TestAuthorize_Flaky503_RecoversOnThirdAttempt(t *testing.T) {
	ts, _ := setupTestServer(t)
	refID := "ref-flaky-004"
	payload := buildRequestPayload(domain.PANFlakyAcquirer, refID, 15000)
	bodyBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	sendAttempt := func() *http.Response {
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/authorizations", bytes.NewReader(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		return resp
	}

	// Attempt 1: 503
	resp1 := sendAttempt()
	assert.Equal(t, http.StatusServiceUnavailable, resp1.StatusCode)
	resp1.Body.Close()

	// Attempt 2: 503
	resp2 := sendAttempt()
	assert.Equal(t, http.StatusServiceUnavailable, resp2.StatusCode)
	resp2.Body.Close()

	// Attempt 3: 200 OK (APPROVED)
	resp3 := sendAttempt()
	defer resp3.Body.Close()
	assert.Equal(t, http.StatusOK, resp3.StatusCode)

	var authResp domain.AuthorizationResponse
	require.NoError(t, json.NewDecoder(resp3.Body).Decode(&authResp))
	assert.Equal(t, "APPROVED", authResp.Status)
	assert.Equal(t, "00", authResp.Code)
	assert.NotEmpty(t, authResp.AuthorizationCode)
}

func TestAuthorize_TimeoutNoAuth_RecoveryReturnsNotFound(t *testing.T) {
	ts, _ := setupTestServer(t)
	refID := "ref-timeout-no-auth-005"
	payload := buildRequestPayload(domain.PANTimeoutNoAuth, refID, 30000)

	bodyBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/authorizations", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)

	// Reconcile via GET: must return 404 (safe, no charge occurred)
	getReq, err := http.NewRequest(http.MethodGet, ts.URL+"/authorizations/"+refID, nil)
	require.NoError(t, err)
	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()

	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
	var errResp domain.ErrorResponse
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&errResp))
	assert.Equal(t, "authorization_not_found", errResp.Error)
}

func TestAuthorize_TimeoutWithAuth_RecoveryReturnsApproved(t *testing.T) {
	ts, _ := setupTestServer(t)
	refID := "ref-timeout-with-auth-006"
	payload := buildRequestPayload(domain.PANTimeoutWithAuth, refID, 45000)

	bodyBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/authorizations", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)

	// Reconcile via GET: must return 200 OK APPROVED (preventing duplicate charge)
	getReq, err := http.NewRequest(http.MethodGet, ts.URL+"/authorizations/"+refID, nil)
	require.NoError(t, err)
	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()

	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	var authResp domain.AuthorizationResponse
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&authResp))
	assert.Equal(t, refID, authResp.ReferenceID)
	assert.Equal(t, "APPROVED", authResp.Status)
	assert.Equal(t, "00", authResp.Code)
	assert.NotEmpty(t, authResp.AuthorizationCode)
}

func TestAuthorize_DefaultCardApproved(t *testing.T) {
	ts, _ := setupTestServer(t)
	payload := buildRequestPayload("4532000000001234", "ref-default-007", 12000)

	bodyBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/authorizations", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var authResp domain.AuthorizationResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&authResp))
	assert.Equal(t, "APPROVED", authResp.Status)
	assert.Equal(t, "00", authResp.Code)
}

func TestAuthorize_ValidationErrors(t *testing.T) {
	ts, _ := setupTestServer(t)

	// Bad JSON
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/authorizations", bytes.NewReader([]byte("{invalid-json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()

	// Missing reference_id
	payload := buildRequestPayload(domain.PANApproved, "", 1000)
	bodyBytes, _ := json.Marshal(payload)
	req2, _ := http.NewRequest(http.MethodPost, ts.URL+"/authorizations", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, resp2.StatusCode)
	resp2.Body.Close()

	// Negative amount
	payloadNeg := buildRequestPayload(domain.PANApproved, "ref-neg", -50)
	bodyBytesNeg, _ := json.Marshal(payloadNeg)
	req3, _ := http.NewRequest(http.MethodPost, ts.URL+"/authorizations", bytes.NewReader(bodyBytesNeg))
	req3.Header.Set("Content-Type", "application/json")
	resp3, err := http.DefaultClient.Do(req3)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, resp3.StatusCode)
	resp3.Body.Close()
}

func TestPCI_Masking(t *testing.T) {
	card := domain.CardPayload{
		Number:      "4242424242424242",
		ExpiryMonth: 12,
		ExpiryYear:  2028,
		CVV:         "987",
		HolderName:  "John Doe",
	}

	str := card.String()
	assert.NotContains(t, str, "4242424242424242")
	assert.NotContains(t, str, "987")
	assert.Contains(t, str, "424242******4242")

	masked := domain.MaskPAN("4242424242424242")
	assert.Equal(t, "424242******4242", masked)
}
