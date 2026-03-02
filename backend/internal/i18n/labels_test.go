package i18n

import "testing"

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		name   string
		status string
		lang   Lang
		want   string
	}{
		{name: "available en", status: "available", lang: LangEN, want: "Available"},
		{name: "available ja", status: "available", lang: LangJA, want: "販売中"},
		{name: "available zh-TW", status: "available", lang: LangZhTW, want: "可購買"},

		{name: "sold en", status: "sold", lang: LangEN, want: "Sold"},
		{name: "sold ja", status: "sold", lang: LangJA, want: "売り切れ"},
		{name: "sold zh-TW", status: "sold", lang: LangZhTW, want: "已售出"},

		{name: "reserved en", status: "reserved", lang: LangEN, want: "Reserved"},
		{name: "reserved ja", status: "reserved", lang: LangJA, want: "取り置き中"},
		{name: "reserved zh-TW", status: "reserved", lang: LangZhTW, want: "已預約"},

		{name: "delisted en", status: "delisted", lang: LangEN, want: "Delisted"},
		{name: "delisted ja", status: "delisted", lang: LangJA, want: "掲載終了"},
		{name: "delisted zh-TW", status: "delisted", lang: LangZhTW, want: "已下架"},

		{name: "unknown passthrough", status: "mystery", lang: LangEN, want: "mystery"},
		{name: "unknown passthrough ja", status: "mystery", lang: LangJA, want: "mystery"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusLabel(tt.status, tt.lang)
			if got != tt.want {
				t.Errorf("StatusLabel(%q, %q) = %q, want %q", tt.status, tt.lang, got, tt.want)
			}
		})
	}
}

func TestConditionLabel(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		lang      Lang
		want      string
	}{
		{name: "new en", condition: "new", lang: LangEN, want: "New"},
		{name: "new ja", condition: "new", lang: LangJA, want: "新品"},
		{name: "new zh-TW", condition: "new", lang: LangZhTW, want: "全新"},

		{name: "like_new en", condition: "like_new", lang: LangEN, want: "Like New"},
		{name: "like_new ja", condition: "like_new", lang: LangJA, want: "未使用に近い"},
		{name: "like_new zh-TW", condition: "like_new", lang: LangZhTW, want: "近全新"},

		{name: "good en", condition: "good", lang: LangEN, want: "Good"},
		{name: "good ja", condition: "good", lang: LangJA, want: "良好"},
		{name: "good zh-TW", condition: "good", lang: LangZhTW, want: "良好"},

		{name: "fair en", condition: "fair", lang: LangEN, want: "Fair"},
		{name: "fair ja", condition: "fair", lang: LangJA, want: "やや傷あり"},
		{name: "fair zh-TW", condition: "fair", lang: LangZhTW, want: "尚可"},

		{name: "poor en", condition: "poor", lang: LangEN, want: "Poor"},
		{name: "poor ja", condition: "poor", lang: LangJA, want: "傷あり"},
		{name: "poor zh-TW", condition: "poor", lang: LangZhTW, want: "較差"},

		{name: "unknown passthrough", condition: "broken", lang: LangEN, want: "broken"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConditionLabel(tt.condition, tt.lang)
			if got != tt.want {
				t.Errorf("ConditionLabel(%q, %q) = %q, want %q", tt.condition, tt.lang, got, tt.want)
			}
		})
	}
}

func TestShippingLabel(t *testing.T) {
	tests := []struct {
		name     string
		shipping string
		lang     Lang
		want     string
	}{
		{name: "free en", shipping: "free", lang: LangEN, want: "Free Shipping"},
		{name: "free ja", shipping: "free", lang: LangJA, want: "送料無料"},
		{name: "free zh-TW", shipping: "free", lang: LangZhTW, want: "免運費"},

		{name: "buyer_pays en", shipping: "buyer_pays", lang: LangEN, want: "Buyer Pays"},
		{name: "buyer_pays ja", shipping: "buyer_pays", lang: LangJA, want: "送料購入者負担"},
		{name: "buyer_pays zh-TW", shipping: "buyer_pays", lang: LangZhTW, want: "買家自付"},

		{name: "included en", shipping: "included", lang: LangEN, want: "Included"},
		{name: "included ja", shipping: "included", lang: LangJA, want: "送料込み"},
		{name: "included zh-TW", shipping: "included", lang: LangZhTW, want: "含運費"},

		{name: "unknown passthrough", shipping: "express", lang: LangEN, want: "express"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShippingLabel(tt.shipping, tt.lang)
			if got != tt.want {
				t.Errorf("ShippingLabel(%q, %q) = %q, want %q", tt.shipping, tt.lang, got, tt.want)
			}
		})
	}
}

func TestContentRatingLabel(t *testing.T) {
	tests := []struct {
		name   string
		rating string
		lang   Lang
		want   string
	}{
		{name: "general en", rating: "general", lang: LangEN, want: "General"},
		{name: "general ja", rating: "general", lang: LangJA, want: "一般"},
		{name: "general zh-TW", rating: "general", lang: LangZhTW, want: "一般"},

		{name: "r18 en", rating: "r18", lang: LangEN, want: "R18"},
		{name: "r18 ja", rating: "r18", lang: LangJA, want: "R18"},
		{name: "r18 zh-TW", rating: "r18", lang: LangZhTW, want: "R18"},

		{name: "unknown passthrough", rating: "pg13", lang: LangEN, want: "pg13"},
		{name: "unknown passthrough ja", rating: "pg13", lang: LangJA, want: "pg13"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContentRatingLabel(tt.rating, tt.lang)
			if got != tt.want {
				t.Errorf("ContentRatingLabel(%q, %q) = %q, want %q", tt.rating, tt.lang, got, tt.want)
			}
		})
	}
}
