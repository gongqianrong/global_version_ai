from __future__ import annotations

from app.translation import TranslationResult, TranslationService


class TestDetectLanguage:
    """Tests for TranslationService.detect_language()."""

    def setup_method(self) -> None:
        self.service = TranslationService()

    def test_detect_japanese_katakana(self) -> None:
        """Katakana text is detected as Japanese."""
        result = self.service.detect_language("グッチ バッグ")
        assert result == "ja"

    def test_detect_english(self) -> None:
        """ASCII-only text is detected as English."""
        result = self.service.detect_language("gucci bag")
        assert result == "en"

    def test_detect_chinese(self) -> None:
        """CJK ideographs without kana are detected as Chinese."""
        result = self.service.detect_language("包包 红色")
        assert result == "zh"

    def test_detect_japanese_hiragana(self) -> None:
        """Hiragana text is detected as Japanese."""
        result = self.service.detect_language("かばん")
        assert result == "ja"

    def test_detect_japanese_mixed_kana_and_kanji(self) -> None:
        """Mixed kana and kanji text is detected as Japanese (not Chinese)."""
        result = self.service.detect_language("赤い鞄")
        assert result == "ja"


class TestTranslateKeyword:
    """Tests for TranslationService.translate_keyword()."""

    def test_translate_english_with_brand_mapping(self) -> None:
        """English keyword with a known brand is translated via mapping."""
        service = TranslationService(brand_mappings={"gucci": "グッチ"})
        result = service.translate_keyword("gucci bag", source_lang="en")

        assert "グッチ" in result.keyword_ja
        assert result.original == "gucci bag"
        assert result.source_lang == "en"
        assert result.was_translated is True

    def test_japanese_keyword_passes_through(self) -> None:
        """Japanese keyword is returned as-is with was_translated=False."""
        service = TranslationService()
        result = service.translate_keyword("グッチ バッグ", source_lang="ja")

        assert result.was_translated is False
        assert result.keyword_ja == "グッチ バッグ"
        assert result.source_lang == "ja"

    def test_auto_detect_language_for_japanese(self) -> None:
        """source_lang is auto-detected when not provided."""
        service = TranslationService()
        result = service.translate_keyword("グッチ バッグ")

        assert result.was_translated is False
        assert result.source_lang == "ja"

    def test_brand_mapping_is_case_insensitive(self) -> None:
        """Brand mappings work regardless of input case."""
        service = TranslationService(brand_mappings={"Gucci": "グッチ"})
        result = service.translate_keyword("GUCCI bag", source_lang="en")

        assert "グッチ" in result.keyword_ja

    def test_unmapped_tokens_pass_through(self) -> None:
        """Tokens without a brand mapping are kept as-is (LLM stub)."""
        service = TranslationService(brand_mappings={"gucci": "グッチ"})
        result = service.translate_keyword("gucci bag", source_lang="en")

        # "bag" has no mapping, so it passes through unchanged
        assert "bag" in result.keyword_ja

    def test_no_brand_mappings(self) -> None:
        """Service works with no brand mappings (empty dict)."""
        service = TranslationService()
        result = service.translate_keyword("gucci bag", source_lang="en")

        assert result.was_translated is True
        assert result.keyword_ja == "gucci bag"


class TestTranslationResult:
    """Tests for the TranslationResult dataclass."""

    def test_dataclass_fields(self) -> None:
        """TranslationResult has the expected fields."""
        result = TranslationResult(
            keyword_ja="グッチ",
            original="gucci",
            source_lang="en",
            was_translated=True,
        )
        assert result.keyword_ja == "グッチ"
        assert result.original == "gucci"
        assert result.source_lang == "en"
        assert result.was_translated is True
