package domain

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// GenerateOrderNumber
// ---------------------------------------------------------------------------

func TestGenerateOrderNumber(t *testing.T) {
	t.Run("starts with RO", func(t *testing.T) {
		n := GenerateOrderNumber()
		if !strings.HasPrefix(n, "RO") {
			t.Errorf("expected prefix RO, got %q", n)
		}
	})

	t.Run("minimum length", func(t *testing.T) {
		// "RO" (2) + timestamp "20060102150405" (14) + 4 hex chars (4) = 20
		n := GenerateOrderNumber()
		if len(n) < 18 {
			t.Errorf("expected length >= 18, got %d (%q)", len(n), n)
		}
	})

	t.Run("two calls return different values", func(t *testing.T) {
		a := GenerateOrderNumber()
		b := GenerateOrderNumber()
		if a == b {
			t.Errorf("expected different order numbers, both were %q", a)
		}
	})
}

// ---------------------------------------------------------------------------
// IsSupportedPayMethod
// ---------------------------------------------------------------------------

func TestIsSupportedPayMethod(t *testing.T) {
	supportedCases := []struct {
		name   string
		method string
	}{
		{"wechat_pay", PayMethodWechat},
		{"alipay", PayMethodAlipay},
		{"apple_pay", PayMethodApplePay},
		{"google_pay", PayMethodGooglePay},
		{"paypal", PayMethodPayPal},
		{"momo", PayMethodMoMo},
		{"zalopay", PayMethodZaloPay},
	}

	for _, tc := range supportedCases {
		tc := tc
		t.Run("supported_"+tc.name, func(t *testing.T) {
			if !IsSupportedPayMethod(tc.method) {
				t.Errorf("expected %q to be supported, but got false", tc.method)
			}
		})
	}

	unsupportedCases := []struct {
		name   string
		method string
	}{
		{"unknown_bitcoin", "bitcoin"},
		{"empty_string", ""},
		{"kakao_pay_not_in_slice", PayMethodKakaoPay},
	}

	for _, tc := range unsupportedCases {
		tc := tc
		t.Run("unsupported_"+tc.name, func(t *testing.T) {
			if IsSupportedPayMethod(tc.method) {
				t.Errorf("expected %q to be unsupported, but got true", tc.method)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GenerateRechargeNo
// ---------------------------------------------------------------------------

func TestGenerateRechargeNo(t *testing.T) {
	t.Run("starts with RC", func(t *testing.T) {
		n := GenerateRechargeNo()
		if !strings.HasPrefix(n, "RC") {
			t.Errorf("expected prefix RC, got %q", n)
		}
	})

	t.Run("minimum length", func(t *testing.T) {
		// "RC" (2) + timestamp (14) + 4 hex chars (4) = 20
		n := GenerateRechargeNo()
		if len(n) < 18 {
			t.Errorf("expected length >= 18, got %d (%q)", len(n), n)
		}
	})

	t.Run("two calls return different values", func(t *testing.T) {
		a := GenerateRechargeNo()
		b := GenerateRechargeNo()
		if a == b {
			t.Errorf("expected different recharge numbers, both were %q", a)
		}
	})
}

// ---------------------------------------------------------------------------
// WaybillStateLabel
// ---------------------------------------------------------------------------

func TestWaybillStateLabel(t *testing.T) {
	cases := []struct {
		state    int
		expected string
	}{
		{WaybillStatePendingConsolidation, "待合单"},
		{WaybillStatePendingPacking, "待打包"},
		{WaybillStatePendingPayment, "待支付"},
		{WaybillStatePendingDispatch, "待出库"},
		{WaybillStateShipped, "已发货"},
		{WaybillStateDelivered, "已收货"},
		{99, "未知"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.expected, func(t *testing.T) {
			got := WaybillStateLabel(tc.state)
			if got != tc.expected {
				t.Errorf("WaybillStateLabel(%d): expected %q, got %q", tc.state, tc.expected, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GenerateWaybillNo
// ---------------------------------------------------------------------------

func TestGenerateWaybillNo(t *testing.T) {
	t.Run("starts with LO", func(t *testing.T) {
		n := GenerateWaybillNo()
		if !strings.HasPrefix(n, "LO") {
			t.Errorf("expected prefix LO, got %q", n)
		}
	})

	t.Run("minimum length", func(t *testing.T) {
		// "LO" (2) + timestamp (14) + 4 hex chars (4) = 20
		n := GenerateWaybillNo()
		if len(n) < 18 {
			t.Errorf("expected length >= 18, got %d (%q)", len(n), n)
		}
	})

	t.Run("two calls return different values", func(t *testing.T) {
		a := GenerateWaybillNo()
		b := GenerateWaybillNo()
		if a == b {
			t.Errorf("expected different waybill numbers, both were %q", a)
		}
	})
}
