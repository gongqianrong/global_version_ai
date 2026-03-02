package i18n

import "testing"

func TestErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		code int
		lang Lang
		want string
	}{
		// 40001 - keyword blocked
		{name: "40001 en", code: 40001, lang: LangEN, want: "keyword is blocked by content policy"},
		{name: "40001 ja", code: 40001, lang: LangJA, want: "キーワードはコンテンツポリシーにより制限されています"},
		{name: "40001 zh-TW", code: 40001, lang: LangZhTW, want: "關鍵字被內容政策封鎖"},

		// 40002 - missing required parameter
		{name: "40002 en", code: 40002, lang: LangEN, want: "missing required parameter"},
		{name: "40002 ja", code: 40002, lang: LangJA, want: "必須パラメータが不足しています"},
		{name: "40002 zh-TW", code: 40002, lang: LangZhTW, want: "缺少必要參數"},

		// 40401 - product not found
		{name: "40401 en", code: 40401, lang: LangEN, want: "product not found"},
		{name: "40401 ja", code: 40401, lang: LangJA, want: "商品が見つかりません"},
		{name: "40401 zh-TW", code: 40401, lang: LangZhTW, want: "找不到商品"},

		// 50001 - internal server error
		{name: "50001 en", code: 50001, lang: LangEN, want: "internal server error"},
		{name: "50001 ja", code: 50001, lang: LangJA, want: "内部サーバーエラー"},
		{name: "50001 zh-TW", code: 50001, lang: LangZhTW, want: "內部伺服器錯誤"},

		// 50003 - service unavailable
		{name: "50003 en", code: 50003, lang: LangEN, want: "service unavailable"},
		{name: "50003 ja", code: 50003, lang: LangJA, want: "サービスが利用できません"},
		{name: "50003 zh-TW", code: 50003, lang: LangZhTW, want: "服務暫時無法使用"},

		// Unmapped code
		{name: "99999 en returns unknown", code: 99999, lang: LangEN, want: "unknown error"},
		{name: "99999 ja returns unknown", code: 99999, lang: LangJA, want: "不明なエラー"},
		{name: "99999 zh-TW returns unknown", code: 99999, lang: LangZhTW, want: "未知錯誤"},

		// Another unmapped code
		{name: "0 en returns unknown", code: 0, lang: LangEN, want: "unknown error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ErrorMessage(tt.code, tt.lang)
			if got != tt.want {
				t.Errorf("ErrorMessage(%d, %q) = %q, want %q", tt.code, tt.lang, got, tt.want)
			}
		})
	}
}
