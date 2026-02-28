package address

import (
	"errors"
	"testing"
)

func validAddress() ShippingAddress {
	return ShippingAddress{
		RecipientName: "田中太郎",
		Phone:         "+81-90-1234-5678",
		CountryCode:   "JP",
		PostalCode:    "100-0001",
		Prefecture:    "東京都",
		City:          "千代田区",
		AddressLine1:  "丸の内1-1-1",
	}
}

func TestValidate_ValidJPAddress(t *testing.T) {
	v := NewValidator()
	addr := validAddress()
	if err := v.Validate(addr); err != nil {
		t.Errorf("expected valid JP address to pass, got %v", err)
	}
}

func TestValidate_ValidTWAddress(t *testing.T) {
	v := NewValidator()
	addr := ShippingAddress{
		RecipientName: "王小明",
		Phone:         "+886-912-345-678",
		CountryCode:   "TW",
		PostalCode:    "100",
		Prefecture:    "台北市",
		City:          "中正區",
		AddressLine1:  "中山南路21號",
	}
	if err := v.Validate(addr); err != nil {
		t.Errorf("expected valid TW address to pass, got %v", err)
	}
}

func TestValidate_BlockedCN(t *testing.T) {
	v := NewValidator()
	addr := validAddress()
	addr.CountryCode = "CN"
	err := v.Validate(addr)
	if !errors.Is(err, ErrBlockedCountry) {
		t.Errorf("expected ErrBlockedCountry for CN, got %v", err)
	}
}

func TestValidate_BlockedCN_Lowercase(t *testing.T) {
	v := NewValidator()
	addr := validAddress()
	addr.CountryCode = "cn"
	err := v.Validate(addr)
	if !errors.Is(err, ErrBlockedCountry) {
		t.Errorf("expected ErrBlockedCountry for 'cn' (lowercase), got %v", err)
	}
}

func TestValidate_BlockedPhone86(t *testing.T) {
	v := NewValidator()
	addr := validAddress()
	addr.Phone = "+86-138-0000-0000"
	err := v.Validate(addr)
	if !errors.Is(err, ErrBlockedPhone) {
		t.Errorf("expected ErrBlockedPhone for +86 prefix, got %v", err)
	}
}

func TestValidate_MissingRecipientName(t *testing.T) {
	v := NewValidator()
	addr := validAddress()
	addr.RecipientName = ""
	err := v.Validate(addr)
	if !errors.Is(err, ErrMissingField) {
		t.Errorf("expected ErrMissingField for empty RecipientName, got %v", err)
	}
}

func TestValidate_MissingPhone(t *testing.T) {
	v := NewValidator()
	addr := validAddress()
	addr.Phone = ""
	err := v.Validate(addr)
	if !errors.Is(err, ErrMissingField) {
		t.Errorf("expected ErrMissingField for empty Phone, got %v", err)
	}
}

func TestValidate_MissingCountryCode(t *testing.T) {
	v := NewValidator()
	addr := validAddress()
	addr.CountryCode = ""
	err := v.Validate(addr)
	if !errors.Is(err, ErrMissingField) {
		t.Errorf("expected ErrMissingField for empty CountryCode, got %v", err)
	}
}

func TestValidate_MissingCity(t *testing.T) {
	v := NewValidator()
	addr := validAddress()
	addr.City = ""
	err := v.Validate(addr)
	if !errors.Is(err, ErrMissingField) {
		t.Errorf("expected ErrMissingField for empty City, got %v", err)
	}
}

func TestValidate_MissingAddressLine1(t *testing.T) {
	v := NewValidator()
	addr := validAddress()
	addr.AddressLine1 = ""
	err := v.Validate(addr)
	if !errors.Is(err, ErrMissingField) {
		t.Errorf("expected ErrMissingField for empty AddressLine1, got %v", err)
	}
}

func TestValidate_WhitespaceOnlyFields(t *testing.T) {
	v := NewValidator()
	addr := validAddress()
	addr.RecipientName = "   "
	err := v.Validate(addr)
	if !errors.Is(err, ErrMissingField) {
		t.Errorf("expected ErrMissingField for whitespace-only RecipientName, got %v", err)
	}
}
