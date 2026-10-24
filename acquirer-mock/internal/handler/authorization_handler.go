package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ImSoHappi/test-haulmer/acquirer-mock/internal/domain"
	"github.com/ImSoHappi/test-haulmer/acquirer-mock/internal/storage"
)

// AuthorizationHandler manages HTTP requests for mock authorizations.
type AuthorizationHandler struct {
	store        *storage.MemoryStore
	timeoutDelay time.Duration
	logger       *slog.Logger
}

// Option configures the AuthorizationHandler.
type Option func(*AuthorizationHandler)

// WithTimeoutDelay sets a custom sleep duration for timeout simulation.
func WithTimeoutDelay(d time.Duration) Option {
	return func(h *AuthorizationHandler) {
		h.timeoutDelay = d
	}
}

// WithLogger sets a custom structured logger.
func WithLogger(logger *slog.Logger) Option {
	return func(h *AuthorizationHandler) {
		h.logger = logger
	}
}

// NewAuthorizationHandler creates a new handler with options.
func NewAuthorizationHandler(store *storage.MemoryStore, opts ...Option) *AuthorizationHandler {
	h := &AuthorizationHandler{
		store:        store,
		timeoutDelay: 3500 * time.Millisecond,
		logger:       slog.Default(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// RegisterRoutes registers the mock endpoints to a standard http.ServeMux.
func (h *AuthorizationHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"UP"}`))
	})
	mux.HandleFunc("POST /authorizations", h.HandleAuthorize)
	mux.HandleFunc("GET /authorizations/{reference_id}", h.HandleGetAuthorization)
}

// HandleAuthorize processes incoming authorization requests according to deterministic PAN behaviors.
func (h *AuthorizationHandler) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	correlationID := r.Header.Get("X-Correlation-ID")
	if correlationID == "" {
		correlationID = uuid.New().String()
	}
	w.Header().Set("X-Correlation-ID", correlationID)
	w.Header().Set("Content-Type", "application/json")

	var req domain.AuthorizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_payload", "Malformed JSON request body")
		return
	}

	if req.ReferenceID == "" || req.Amount <= 0 || req.Currency != "CLP" || req.Card.Number == "" {
		h.writeError(w, http.StatusUnprocessableEntity, "validation_error", "Missing or invalid required authorization fields")
		return
	}

	behavior := domain.ClassifyCard(req.Card.Number)
	maskedPAN := domain.MaskPAN(req.Card.Number)

	h.logger.Info("received authorization request",
		"correlation_id", correlationID,
		"reference_id", req.ReferenceID,
		"amount", req.Amount,
		"currency", req.Currency,
		"masked_pan", maskedPAN,
		"behavior", string(behavior),
	)

	switch behavior {
	case domain.BehaviorApproved:
		h.respondApproved(w, req)

	case domain.BehaviorDeclinedInsufficientFunds:
		h.respondDeclined(w, req, "51", "insufficient_funds", "Your card has insufficient funds.")

	case domain.BehaviorPersistent503:
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "The payment provider encountered an unexpected system outage.")

	case domain.BehaviorFlaky503:
		attempts := h.store.IncrementAttempts(req.ReferenceID)
		if attempts <= 2 {
			h.logger.Warn("flaky acquirer simulating 503 outage",
				"correlation_id", correlationID,
				"reference_id", req.ReferenceID,
				"attempt", attempts,
			)
			h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "The payment provider is experiencing high load. Please retry.")
			return
		}
		h.logger.Info("flaky acquirer recovered on retry attempt",
			"correlation_id", correlationID,
			"reference_id", req.ReferenceID,
			"attempt", attempts,
		)
		h.respondApproved(w, req)

	case domain.BehaviorTimeoutNoAuth:
		h.logger.Warn("simulating timeout without authorization",
			"correlation_id", correlationID,
			"reference_id", req.ReferenceID,
			"delay", h.timeoutDelay,
		)
		time.Sleep(h.timeoutDelay)
		h.writeError(w, http.StatusGatewayTimeout, "gateway_timeout", "Acquirer upstream processing timed out")

	case domain.BehaviorTimeoutWithAuth:
		h.logger.Warn("simulating timeout with prior authorization recorded",
			"correlation_id", correlationID,
			"reference_id", req.ReferenceID,
			"delay", h.timeoutDelay,
		)
		// Register authorization first, then delay
		resp := domain.AuthorizationResponse{
			ReferenceID:       req.ReferenceID,
			Status:            "APPROVED",
			Code:              "00",
			AuthorizationCode: fmt.Sprintf("AUT-%s", strings.ToUpper(uuid.New().String()[:8])),
			Amount:            req.Amount,
			Currency:          req.Currency,
			CreatedAt:         time.Now().UTC(),
		}
		h.store.SaveAuthorization(resp)

		time.Sleep(h.timeoutDelay)
		h.writeError(w, http.StatusGatewayTimeout, "gateway_timeout", "Acquirer response delayed")
	}
}

// HandleGetAuthorization retrieves an authorization by its reference ID.
func (h *AuthorizationHandler) HandleGetAuthorization(w http.ResponseWriter, r *http.Request) {
	referenceID := r.PathValue("reference_id")
	if referenceID == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_reference_id", "Reference ID must not be empty")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp, exists := h.store.GetAuthorization(referenceID)
	if !exists {
		h.writeError(w, http.StatusNotFound, "authorization_not_found", "No authorization recorded for this reference_id")
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AuthorizationHandler) respondApproved(w http.ResponseWriter, req domain.AuthorizationRequest) {
	resp := domain.AuthorizationResponse{
		ReferenceID:       req.ReferenceID,
		Status:            "APPROVED",
		Code:              "00",
		AuthorizationCode: fmt.Sprintf("AUT-%s", strings.ToUpper(uuid.New().String()[:8])),
		Amount:            req.Amount,
		Currency:          req.Currency,
		CreatedAt:         time.Now().UTC(),
	}
	h.store.SaveAuthorization(resp)

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AuthorizationHandler) respondDeclined(w http.ResponseWriter, req domain.AuthorizationRequest, code, declineCode, message string) {
	resp := domain.AuthorizationResponse{
		ReferenceID: req.ReferenceID,
		Status:      "DECLINED",
		Code:        code,
		DeclineCode: declineCode,
		Message:     message,
		Amount:      req.Amount,
		Currency:    req.Currency,
		CreatedAt:   time.Now().UTC(),
	}
	h.store.SaveAuthorization(resp)

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AuthorizationHandler) writeError(w http.ResponseWriter, status int, errCode, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(domain.ErrorResponse{
		Error:   errCode,
		Message: msg,
	})
}
