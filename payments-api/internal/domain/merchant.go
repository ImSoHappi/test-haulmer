package domain

// MerchantStatus represents the operational status of a registered merchant.
type MerchantStatus string

const (
	MerchantStatusActive   MerchantStatus = "ACTIVE"
	MerchantStatusInactive MerchantStatus = "INACTIVE"
)

// Merchant represents a registered commercial account in the payments system.
type Merchant struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Status MerchantStatus `json:"status"`
}

// NewMerchant constructs a Merchant entity.
func NewMerchant(id, name string, status MerchantStatus) Merchant {
	return Merchant{
		ID:     id,
		Name:   name,
		Status: status,
	}
}

// IsActive returns true if the merchant is active and permitted to process transactions.
func (m Merchant) IsActive() bool {
	return m.Status == MerchantStatusActive
}
