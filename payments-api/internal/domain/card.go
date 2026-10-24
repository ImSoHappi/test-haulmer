package domain

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"
)

// CardNumber is a value object representing a validated PAN.
type CardNumber struct {
	pan string
}

// NewCardNumber sanitizes, checks length (13-19 digits) and validates the Luhn checksum.
func NewCardNumber(raw string) (CardNumber, error) {
	// Strip spaces and dashes
	clean := strings.ReplaceAll(strings.ReplaceAll(raw, " ", ""), "-", "")

	if len(clean) < 13 || len(clean) > 19 {
		return CardNumber{}, ErrInvalidCardFormat
	}

	for _, ch := range clean {
		if !unicode.IsDigit(ch) {
			return CardNumber{}, ErrInvalidCardFormat
		}
	}

	if !validateLuhn(clean) {
		return CardNumber{}, ErrInvalidCardLuhn
	}

	return CardNumber{pan: clean}, nil
}

// BIN returns the first 6 digits of the card number (Bank Identification Number).
func (c CardNumber) BIN() string {
	if len(c.pan) >= 6 {
		return c.pan[:6]
	}
	return ""
}

// Last4 returns the last 4 digits of the card number.
func (c CardNumber) Last4() string {
	if len(c.pan) >= 4 {
		return c.pan[len(c.pan)-4:]
	}
	return ""
}

// Mask returns the PCI-safe masked PAN representation (BIN + ****** + Last4).
func (c CardNumber) Mask() string {
	if len(c.pan) < 10 {
		return "******"
	}
	return c.pan[:6] + "******" + c.pan[len(c.pan)-4:]
}

// String implements fmt.Stringer to prevent raw PAN leaks in prints.
func (c CardNumber) String() string {
	return c.Mask()
}

// LogValue implements slog.LogValuer to prevent raw PAN leaks in structured logs.
func (c CardNumber) LogValue() slog.Value {
	return slog.StringValue(c.Mask())
}

// Raw returns the full PAN. It must ONLY be called when preparing the outbound acquirer payload.
func (c CardNumber) Raw() string {
	return c.pan
}

// validateLuhn implements the standard Luhn Modulo 10 checksum algorithm.
func validateLuhn(number string) bool {
	var sum int
	alternate := false

	for i := len(number) - 1; i >= 0; i-- {
		n := int(number[i] - '0')
		if alternate {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alternate = !alternate
	}

	return sum%10 == 0
}

// CVV is a value object representing the card security verification code.
type CVV struct {
	code string
}

// NewCVV validates that the code consists of 3 or 4 digits.
func NewCVV(raw string) (CVV, error) {
	if len(raw) < 3 || len(raw) > 4 {
		return CVV{}, ErrInvalidCVV
	}
	for _, ch := range raw {
		if !unicode.IsDigit(ch) {
			return CVV{}, ErrInvalidCVV
		}
	}
	return CVV{code: raw}, nil
}

// String implements fmt.Stringer to ensure CVV is never exposed in text prints.
func (c CVV) String() string {
	return "[REDACTED]"
}

// LogValue implements slog.LogValuer to ensure CVV is never exposed in structured logs.
func (c CVV) LogValue() slog.Value {
	return slog.StringValue("[REDACTED]")
}

// Raw returns the raw CVV. It must ONLY be called when preparing the outbound acquirer payload.
func (c CVV) Raw() string {
	return c.code
}

// CardExpiry is a value object representing the card expiration date.
type CardExpiry struct {
	Month int
	Year  int
}

// NewCardExpiry validates the month and normalizes two-digit years (e.g. 28 -> 2028).
func NewCardExpiry(month, year int) (CardExpiry, error) {
	if month < 1 || month > 12 {
		return CardExpiry{}, ErrCardExpired
	}
	if year < 100 {
		year += 2000
	}
	return CardExpiry{Month: month, Year: year}, nil
}

// IsExpired returns true if the card has expired relative to the provided time.
// Cards are valid through the very last second of the expiration month.
func (e CardExpiry) IsExpired(now time.Time) bool {
	// The first day of the month after expiry
	firstOfNextMonth := time.Date(e.Year, time.Month(e.Month)+1, 1, 0, 0, 0, 0, time.UTC)
	return !now.UTC().Before(firstOfNextMonth)
}

// Card aggregates card details and implements PCI masking.
type Card struct {
	Number     CardNumber
	Expiry     CardExpiry
	CVV        CVV
	HolderName string
}

// NewCard constructs a Card aggregate.
func NewCard(number CardNumber, expiry CardExpiry, cvv CVV, holderName string) Card {
	return Card{
		Number:     number,
		Expiry:     expiry,
		CVV:        cvv,
		HolderName: holderName,
	}
}

// String implements fmt.Stringer.
func (c Card) String() string {
	return fmt.Sprintf("Card{MaskedPAN: %s, Expiry: %02d/%04d, Holder: %s}", c.Number.Mask(), c.Expiry.Month, c.Expiry.Year, c.HolderName)
}

// LogValue implements slog.LogValuer.
func (c Card) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("masked_pan", c.Number.Mask()),
		slog.Int("expiry_month", c.Expiry.Month),
		slog.Int("expiry_year", c.Expiry.Year),
		slog.String("holder_name", c.HolderName),
	)
}
