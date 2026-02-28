// Package address provides shipping address validation for the collection gateway.
package address

import (
	"errors"
	"strings"
)

// ShippingAddress represents a customer's shipping address.
type ShippingAddress struct {
	RecipientName string `json:"recipient_name"`
	Phone         string `json:"phone"`
	CountryCode   string `json:"country_code"` // ISO 3166-1 alpha-2, e.g. "JP", "TW"
	PostalCode    string `json:"postal_code"`
	Prefecture    string `json:"prefecture"`
	City          string `json:"city"`
	AddressLine1  string `json:"address_line1"`
	AddressLine2  string `json:"address_line2,omitempty"`
}

// Validation errors.
var (
	ErrBlockedCountry = errors.New("address: country is blocked")
	ErrBlockedPhone   = errors.New("address: phone prefix is blocked")
	ErrMissingField   = errors.New("address: required field is missing")
)

// Validator validates shipping addresses against regional restrictions.
type Validator struct {
	blockedCountries  map[string]struct{} // uppercase ISO codes
	blockedPhonePfxs  []string
}

// NewValidator creates a Validator with default restrictions:
// blocks CN country code and +86 phone prefix.
func NewValidator() *Validator {
	return &Validator{
		blockedCountries: map[string]struct{}{
			"CN": {},
		},
		blockedPhonePfxs: []string{"+86"},
	}
}

// Validate checks a ShippingAddress for required fields and regional restrictions.
// It returns the first error encountered.
func (v *Validator) Validate(addr ShippingAddress) error {
	// Check required fields.
	if strings.TrimSpace(addr.RecipientName) == "" {
		return ErrMissingField
	}
	if strings.TrimSpace(addr.Phone) == "" {
		return ErrMissingField
	}
	if strings.TrimSpace(addr.CountryCode) == "" {
		return ErrMissingField
	}
	if strings.TrimSpace(addr.PostalCode) == "" {
		return ErrMissingField
	}
	if strings.TrimSpace(addr.City) == "" {
		return ErrMissingField
	}
	if strings.TrimSpace(addr.AddressLine1) == "" {
		return ErrMissingField
	}

	// Check blocked country.
	upper := strings.ToUpper(strings.TrimSpace(addr.CountryCode))
	if _, blocked := v.blockedCountries[upper]; blocked {
		return ErrBlockedCountry
	}

	// Check blocked phone prefix.
	phone := strings.TrimSpace(addr.Phone)
	for _, pfx := range v.blockedPhonePfxs {
		if strings.HasPrefix(phone, pfx) {
			return ErrBlockedPhone
		}
	}

	return nil
}
