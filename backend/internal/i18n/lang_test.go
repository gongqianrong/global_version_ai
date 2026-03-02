package i18n

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   Lang
	}{
		{name: "empty string returns default", header: "", want: LangZhTW},
		{name: "ja returns Japanese", header: "ja", want: LangJA},
		{name: "en returns English", header: "en", want: LangEN},
		{name: "zh-TW returns Traditional Chinese", header: "zh-TW", want: LangZhTW},
		{name: "zh-TW with en fallback", header: "zh-TW,en;q=0.9", want: LangZhTW},
		{name: "en-US matches English", header: "en-US,en;q=0.9", want: LangEN},
		{name: "ja-JP matches Japanese", header: "ja-JP", want: LangJA},
		{name: "unsupported language falls back to default", header: "fr", want: LangZhTW},
		{name: "zh-Hant matches Traditional Chinese", header: "zh-Hant", want: LangZhTW},
		{name: "zh matches Traditional Chinese", header: "zh", want: LangZhTW},
		{name: "case insensitive zh-tw", header: "ZH-TW", want: LangZhTW},
		{name: "case insensitive JA", header: "JA", want: LangJA},
		{name: "case insensitive EN", header: "EN", want: LangEN},
		{name: "multiple with quality first unsupported", header: "fr,ja;q=0.8", want: LangJA},
		{name: "whitespace trimmed", header: " ja , en;q=0.9 ", want: LangJA},
		{name: "wildcard falls back to default", header: "*", want: LangZhTW},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAcceptLanguage(tt.header)
			if got != tt.want {
				t.Errorf("ParseAcceptLanguage(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

func TestFromContext_EmptyContext(t *testing.T) {
	ctx := context.Background()
	got := FromContext(ctx)
	if got != DefaultLang {
		t.Errorf("FromContext(empty) = %q, want %q", got, DefaultLang)
	}
}

func TestWithLangAndFromContext_Roundtrip(t *testing.T) {
	tests := []struct {
		name string
		lang Lang
	}{
		{name: "Japanese", lang: LangJA},
		{name: "English", lang: LangEN},
		{name: "Traditional Chinese", lang: LangZhTW},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithLang(context.Background(), tt.lang)
			got := FromContext(ctx)
			if got != tt.lang {
				t.Errorf("WithLang/FromContext roundtrip = %q, want %q", got, tt.lang)
			}
		})
	}
}

func TestFromRequest(t *testing.T) {
	tests := []struct {
		name       string
		acceptLang string
		want       Lang
	}{
		{name: "no header returns default", acceptLang: "", want: LangZhTW},
		{name: "Japanese header", acceptLang: "ja", want: LangJA},
		{name: "English header", acceptLang: "en-US,en;q=0.9", want: LangEN},
		{name: "Traditional Chinese header", acceptLang: "zh-TW", want: LangZhTW},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.acceptLang != "" {
				req.Header.Set("Accept-Language", tt.acceptLang)
			}
			got := FromRequest(req)
			if got != tt.want {
				t.Errorf("FromRequest(Accept-Language: %q) = %q, want %q", tt.acceptLang, got, tt.want)
			}
		})
	}
}

func TestSupportedLangs(t *testing.T) {
	if len(SupportedLangs) != 3 {
		t.Errorf("SupportedLangs length = %d, want 3", len(SupportedLangs))
	}
	expected := []Lang{LangZhTW, LangJA, LangEN}
	for i, lang := range expected {
		if SupportedLangs[i] != lang {
			t.Errorf("SupportedLangs[%d] = %q, want %q", i, SupportedLangs[i], lang)
		}
	}
}
