package i18n

// langMessages maps a Lang to a translated string.
type langMessages map[Lang]string

// errorMessages maps error codes to their translated messages.
var errorMessages = map[int]langMessages{
	40001: {
		LangEN:   "keyword is blocked by content policy",
		LangJA:   "キーワードはコンテンツポリシーにより制限されています",
		LangZhTW: "關鍵字被內容政策封鎖",
	},
	40002: {
		LangEN:   "missing required parameter",
		LangJA:   "必須パラメータが不足しています",
		LangZhTW: "缺少必要參數",
	},
	40401: {
		LangEN:   "product not found",
		LangJA:   "商品が見つかりません",
		LangZhTW: "找不到商品",
	},
	50001: {
		LangEN:   "internal server error",
		LangJA:   "内部サーバーエラー",
		LangZhTW: "內部伺服器錯誤",
	},
	50003: {
		LangEN:   "service unavailable",
		LangJA:   "サービスが利用できません",
		LangZhTW: "服務暫時無法使用",
	},
}

// unknownError holds the fallback message for unmapped error codes.
var unknownError = langMessages{
	LangEN:   "unknown error",
	LangJA:   "不明なエラー",
	LangZhTW: "未知錯誤",
}

// ErrorMessage returns the translated error message for the given code and
// language. If the code is not mapped, a generic "unknown error" message is
// returned in the requested language.
func ErrorMessage(code int, lang Lang) string {
	if msgs, ok := errorMessages[code]; ok {
		if msg, ok := msgs[lang]; ok {
			return msg
		}
	}
	if msg, ok := unknownError[lang]; ok {
		return msg
	}
	return unknownError[LangEN]
}
