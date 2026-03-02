package i18n

// labelTable maps a key to its translations across supported languages.
type labelTable map[string]langMessages

// labelLookup returns the translated label for key in the given language.
// If the key is not found in the table, it returns the key itself (passthrough).
func labelLookup(table labelTable, key string, lang Lang) string {
	if msgs, ok := table[key]; ok {
		if msg, ok := msgs[lang]; ok {
			return msg
		}
	}
	return key
}

// statusLabels holds translations for product status values.
var statusLabels = labelTable{
	"available": {
		LangEN:   "Available",
		LangJA:   "販売中",
		LangZhTW: "可購買",
	},
	"sold": {
		LangEN:   "Sold",
		LangJA:   "売り切れ",
		LangZhTW: "已售出",
	},
	"reserved": {
		LangEN:   "Reserved",
		LangJA:   "取り置き中",
		LangZhTW: "已預約",
	},
	"delisted": {
		LangEN:   "Delisted",
		LangJA:   "掲載終了",
		LangZhTW: "已下架",
	},
}

// conditionLabels holds translations for product condition values.
var conditionLabels = labelTable{
	"new": {
		LangEN:   "New",
		LangJA:   "新品",
		LangZhTW: "全新",
	},
	"like_new": {
		LangEN:   "Like New",
		LangJA:   "未使用に近い",
		LangZhTW: "近全新",
	},
	"good": {
		LangEN:   "Good",
		LangJA:   "良好",
		LangZhTW: "良好",
	},
	"fair": {
		LangEN:   "Fair",
		LangJA:   "やや傷あり",
		LangZhTW: "尚可",
	},
	"poor": {
		LangEN:   "Poor",
		LangJA:   "傷あり",
		LangZhTW: "較差",
	},
}

// shippingLabels holds translations for shipping type values.
var shippingLabels = labelTable{
	"free": {
		LangEN:   "Free Shipping",
		LangJA:   "送料無料",
		LangZhTW: "免運費",
	},
	"buyer_pays": {
		LangEN:   "Buyer Pays",
		LangJA:   "送料購入者負担",
		LangZhTW: "買家自付",
	},
	"included": {
		LangEN:   "Included",
		LangJA:   "送料込み",
		LangZhTW: "含運費",
	},
}

// contentRatingLabels holds translations for content rating values.
var contentRatingLabels = labelTable{
	"general": {
		LangEN:   "General",
		LangJA:   "一般",
		LangZhTW: "一般",
	},
	"r18": {
		LangEN:   "R18",
		LangJA:   "R18",
		LangZhTW: "R18",
	},
}

// StatusLabel returns the translated label for a product status value.
// Unknown status values are returned as-is.
func StatusLabel(status string, lang Lang) string {
	return labelLookup(statusLabels, status, lang)
}

// ConditionLabel returns the translated label for a product condition value.
// Unknown condition values are returned as-is.
func ConditionLabel(condition string, lang Lang) string {
	return labelLookup(conditionLabels, condition, lang)
}

// ShippingLabel returns the translated label for a shipping type value.
// Unknown shipping values are returned as-is.
func ShippingLabel(shipping string, lang Lang) string {
	return labelLookup(shippingLabels, shipping, lang)
}

// ContentRatingLabel returns the translated label for a content rating value.
// Unknown rating values are returned as-is.
func ContentRatingLabel(rating string, lang Lang) string {
	return labelLookup(contentRatingLabels, rating, lang)
}
