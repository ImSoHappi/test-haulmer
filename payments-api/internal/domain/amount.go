package domain

import "strings"

// CurrencyCLP is the only supported currency in this processing service (BR-01).
const CurrencyCLP = "CLP"

// Amount represents an immutable payment amount in Chilean Pesos (CLP).
// Chilean pesos have no decimals (cents).
type Amount struct {
	value int64
}

// NewAmount validates that the amount is strictly positive (> 0).
func NewAmount(value int64) (Amount, error) {
	if value <= 0 {
		return Amount{}, ErrInvalidAmount
	}
	return Amount{value: value}, nil
}

// Int64 returns the raw integer amount.
func (a Amount) Int64() int64 {
	return a.value
}

// ValidateMax checks that the amount does not exceed the maximum allowable limit.
func (a Amount) ValidateMax(limit int64) error {
	if limit > 0 && a.value > limit {
		return ErrAmountExceedsLimit
	}
	return nil
}

// NewCurrency validates that the provided currency string matches the supported CLP currency.
func NewCurrency(raw string) (string, error) {
	if strings.ToUpper(strings.TrimSpace(raw)) != CurrencyCLP {
		return "", ErrUnsupportedCurrency
	}
	return CurrencyCLP, nil
}
