from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Dict, List, Optional


@dataclass
class TranslationResult:
    """Result of a keyword translation operation."""
    keyword_ja: str
    original: str
    source_lang: str
    was_translated: bool


# Unicode ranges for Japanese/Chinese character detection
# Hiragana: U+3040 - U+309F
# Katakana: U+30A0 - U+30FF
# Katakana Phonetic Extensions: U+31F0 - U+31FF
# CJK Unified Ideographs (Kanji/Hanzi): U+4E00 - U+9FFF
# CJK Unified Ideographs Extension A: U+3400 - U+4DBF
_HIRAGANA_RE = re.compile(r'[\u3040-\u309F]')
_KATAKANA_RE = re.compile(r'[\u30A0-\u30FF\u31F0-\u31FF]')
_CJK_RE = re.compile(r'[\u4E00-\u9FFF\u3400-\u4DBF]')


class TranslationService:
    """Service for detecting language and translating keywords to Japanese.

    Uses regex-based heuristics for language detection and a brand mapping
    dictionary for translating known brand names. Non-brand tokens are passed
    through as-is (a future LLM integration would handle full translation).
    """

    def __init__(self, brand_mappings: Optional[Dict[str, str]] = None) -> None:
        """Initialize the translation service.

        Args:
            brand_mappings: Dictionary mapping brand names (in any language)
                to their Japanese equivalents. Keys are lowercased internally.
        """
        if brand_mappings is None:
            self._brand_mappings = {}  # type: Dict[str, str]
        else:
            self._brand_mappings = {
                k.lower(): v for k, v in brand_mappings.items()
            }

    def detect_language(self, text: str) -> str:
        """Detect the language of the given text using Unicode ranges.

        Detection priority:
        1. If hiragana or katakana characters are present -> "ja"
        2. If CJK ideographs are present (but no kana) -> "zh"
        3. Otherwise -> "en"

        Args:
            text: The text to detect the language of.

        Returns:
            A language code: "ja", "zh", or "en".
        """
        has_hiragana = bool(_HIRAGANA_RE.search(text))
        has_katakana = bool(_KATAKANA_RE.search(text))

        if has_hiragana or has_katakana:
            return "ja"

        has_cjk = bool(_CJK_RE.search(text))
        if has_cjk:
            return "zh"

        return "en"

    def translate_keyword(
        self,
        keyword: str,
        source_lang: Optional[str] = None,
    ) -> TranslationResult:
        """Translate a keyword to Japanese.

        If the keyword is already in Japanese, it is returned as-is with
        was_translated=False. Otherwise, tokens are split by whitespace and
        each token is looked up in the brand mappings. Tokens that are not
        found in the mappings are kept as-is (stub for future LLM translation).

        Args:
            keyword: The keyword to translate.
            source_lang: Optional language code. If not provided, it will be
                auto-detected via detect_language().

        Returns:
            A TranslationResult with the translated keyword and metadata.
        """
        if source_lang is None:
            source_lang = self.detect_language(keyword)

        if source_lang == "ja":
            return TranslationResult(
                keyword_ja=keyword,
                original=keyword,
                source_lang=source_lang,
                was_translated=False,
            )

        # Split into tokens and translate each one
        tokens = keyword.split()
        translated_tokens = []  # type: List[str]

        for token in tokens:
            mapped = self._brand_mappings.get(token.lower())
            if mapped is not None:
                translated_tokens.append(mapped)
            else:
                # Stub: in the future, this would call an LLM for translation.
                # For now, pass the token through unchanged.
                translated_tokens.append(token)

        keyword_ja = " ".join(translated_tokens)

        return TranslationResult(
            keyword_ja=keyword_ja,
            original=keyword,
            source_lang=source_lang,
            was_translated=True,
        )
