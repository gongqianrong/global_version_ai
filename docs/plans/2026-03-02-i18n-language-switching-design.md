# API Response Language Switching Design

## Overview

All API response text switches based on `Accept-Language` header. Supports zh-TW (default), ja, en with extensible structure.

## Two-Layer Translation

| Type | Method | Timing | Performance |
|------|--------|--------|-------------|
| Static text (errors, status enums) | Go map lookup | Compile time | Zero |
| Product content (title, description) | Free/cheap translate API | Pre-translate on ES write | Zero (map lookup on read) |

## Modules

### `internal/i18n` (new)
- `lang.go`: Language constants (zh-TW, ja, en), default language
- `messages.go`: Error code → message translation table
- `labels.go`: Status/condition enum display text
- `context.go`: Read/write language from context

### `internal/translate` (new)
- Wrap free translation API
- Called in output pipeline before ES write
- Pre-translates to all supported languages

### Middleware changes
- New language middleware: parse `Accept-Language` → write to request context

### Response layer changes
- `ErrorWithCode`: auto-translate message by context language
- `Success`: product data selects translated version by language

### Product model
- Reuse existing `TitleTranslated` / `DescTranslated` map[string]string

## Data Flow

```
Request → LangMiddleware(Accept-Language → ctx) → handler → response layer
                                                              ↓
                                                    By ctx lang:
                                                    - Static: i18n map lookup
                                                    - Product: TitleTranslated[lang]
                                                    - No translation: original + is_translated=false
```

```
Indexing: Crawl → translate API (ja→zh-TW, ja→en) → store in ES (all language versions)
```

## Languages
- zh-TW (default), ja, en — extensible
