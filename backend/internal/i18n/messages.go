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
	40003: {
		LangEN:   "invalid parameter value",
		LangJA:   "パラメータの値が無効です",
		LangZhTW: "參數值無效",
	},
	40100: {
		LangEN:   "authentication required",
		LangJA:   "認証が必要です",
		LangZhTW: "需要認證",
	},
	40101: {
		LangEN:   "invalid or expired token",
		LangJA:   "トークンが無効または期限切れです",
		LangZhTW: "無效或過期的令牌",
	},
	40901: {
		LangEN:   "email already registered",
		LangJA:   "このメールアドレスは既に登録されています",
		LangZhTW: "電子郵件已被註冊",
	},
	40004: {
		LangEN:   "invalid email format",
		LangJA:   "メールアドレスの形式が無効です",
		LangZhTW: "電子郵件格式無效",
	},
	40005: {
		LangEN:   "please wait before requesting another code",
		LangJA:   "再送信までお待ちください",
		LangZhTW: "請稍後再請求驗證碼",
	},
	40006: {
		LangEN:   "invalid or expired verification code",
		LangJA:   "認証コードが無効または期限切れです",
		LangZhTW: "驗證碼無效或已過期",
	},
	40007: {
		LangEN:   "invalid OAuth token",
		LangJA:   "OAuthトークンが無効です",
		LangZhTW: "無效的 OAuth 令牌",
	},
	40008: {
		LangEN:   "OAuth provider error",
		LangJA:   "OAuthプロバイダーエラー",
		LangZhTW: "OAuth 提供者錯誤",
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
