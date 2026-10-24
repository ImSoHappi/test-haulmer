package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/http/middleware"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/payment"
)

var (
	numericRegex    = regexp.MustCompile(`^[0-9]+$`)
	orderRefPattern = regexp.MustCompile(`^[A-Za-z0-9]{6}$`)
)

// PaymentService defines the application interface for payment operations.
type PaymentService interface {
	ProcessPayment(ctx context.Context, req payment.ProcessRequest) (*payment.ProcessResponse, error)
	GetPayment(ctx context.Context, req payment.GetRequest) (*payment.GetResponse, error)
	ListPayments(ctx context.Context, req payment.ListRequest) (*payment.ListResponse, error)
}

// PaymentHandler handles HTTP requests for payments.
type PaymentHandler struct {
	service PaymentService
}

// NewPaymentHandler constructs a new PaymentHandler.
func NewPaymentHandler(service PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

type paymentRequestBody struct {
	MerchantID      string `json:"merchant_id"`
	OrderReference  string `json:"order_reference"`
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
	CardNumber      string `json:"card_number"`
	CardExpiryMonth int    `json:"card_expiry_month"`
	CardExpiryYear  int    `json:"card_expiry_year"`
	CardCVV         string `json:"card_cvv"`
	CardHolderName  string `json:"card_holder_name"`
}

type errorResponse struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id"`
}

func respondError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	code := "internal_server_error"
	msg := "An unexpected error occurred while processing the request."

	var domErr *domain.DomainError
	if errors.As(err, &domErr) {
		status = domErr.HTTPStatus
		code = domErr.Code
		msg = domErr.Message
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Code:          code,
		Message:       msg,
		CorrelationID: middleware.GetCorrelationID(r.Context()),
	})
}

// ProcessPayment handles POST /v1/payments.
func (h *PaymentHandler) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idemKey == "" {
		respondError(w, r, domain.ErrMissingRequiredField)
		return
	}

	var body paymentRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, r, domain.ErrMissingRequiredField)
		return
	}

	merchantID := strings.TrimSpace(body.MerchantID)
	if !numericRegex.MatchString(merchantID) {
		respondError(w, r, domain.ErrMerchantNotFound)
		return
	}

	orderRef := strings.TrimSpace(body.OrderReference)
	if !orderRefPattern.MatchString(orderRef) {
		respondError(w, r, domain.ErrMissingRequiredField)
		return
	}

	corrID := middleware.GetCorrelationID(r.Context())

	req := payment.ProcessRequest{
		MerchantID:      merchantID,
		IdempotencyKey:  idemKey,
		OrderReference:  orderRef,
		Amount:          body.Amount,
		Currency:        body.Currency,
		CardNumber:      body.CardNumber,
		CardExpiryMonth: body.CardExpiryMonth,
		CardExpiryYear:  body.CardExpiryYear,
		CardCVV:         body.CardCVV,
		CardHolderName:  body.CardHolderName,
		CorrelationID:   corrID,
	}

	res, err := h.service.ProcessPayment(r.Context(), req)
	if err != nil {
		respondError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if res.IsReplayed {
		w.Header().Set("Idempotent-Replayed", "true")
		w.WriteHeader(http.StatusOK)
	} else {
		switch res.Status {
		case "PENDING", "PROCESSING":
			w.Header().Set("Retry-After", "5")
			w.WriteHeader(http.StatusAccepted)
		case "DECLINED":
			w.WriteHeader(http.StatusPaymentRequired)
		default:
			w.WriteHeader(http.StatusCreated)
		}
	}

	_ = json.NewEncoder(w).Encode(res)
}

// GetPayment handles GET /v1/payments/{id}.
func (h *PaymentHandler) GetPayment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if _, err := uuid.Parse(idStr); err != nil {
		respondError(w, r, domain.ErrTransactionNotFound)
		return
	}

	merchantID := strings.TrimSpace(r.Header.Get("X-Merchant-ID"))

	res, err := h.service.GetPayment(r.Context(), payment.GetRequest{
		TransactionID: idStr,
		MerchantID:    merchantID,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

// ListPayments handles GET /v1/payments.
func (h *PaymentHandler) ListPayments(w http.ResponseWriter, r *http.Request) {
	merchantID := strings.TrimSpace(r.Header.Get("X-Merchant-ID"))
	if merchantID == "" {
		respondError(w, r, domain.ErrMissingRequiredField)
		return
	}

	var statusFilter *domain.TransactionStatus
	if s := strings.TrimSpace(r.URL.Query().Get("status")); s != "" {
		ts := domain.TransactionStatus(strings.ToUpper(s))
		statusFilter = &ts
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	res, err := h.service.ListPayments(r.Context(), payment.ListRequest{
		MerchantID: merchantID,
		Status:     statusFilter,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}
