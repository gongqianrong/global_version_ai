from __future__ import annotations

from app.brand_extractor import BrandExtractor, ExtractionResult, KNOWN_BRANDS


class TestBrandExtractor:
    """Tests for BrandExtractor.extract()."""

    def setup_method(self) -> None:
        self.extractor = BrandExtractor()

    def test_extract_brand_from_title(self) -> None:
        """A known brand in the title is extracted with high confidence."""
        result = self.extractor.extract(
            title="CHANEL シャネル マトラッセ バッグ",
            description="",
            category="",
        )

        assert result is not None
        assert "CHANEL" in result.brand_name or "シャネル" in result.brand_name
        assert result.confidence > 0

    def test_no_brand_in_handmade_item(self) -> None:
        """A handmade item with no brand returns None or low confidence."""
        result = self.extractor.extract(
            title="ハンドメイド トートバッグ",
            description="",
            category="",
        )

        assert result is None or result.confidence < 0.5

    def test_title_match_has_high_confidence(self) -> None:
        """Brand found in title gets 0.95 confidence."""
        result = self.extractor.extract(
            title="GUCCI レザーバッグ",
            description="",
            category="",
        )

        assert result is not None
        assert result.confidence == 0.95

    def test_description_match_has_lower_confidence(self) -> None:
        """Brand found only in description gets 0.75 confidence."""
        result = self.extractor.extract(
            title="レザーバッグ",
            description="GUCCI の正規品です",
            category="",
        )

        assert result is not None
        assert result.confidence == 0.75

    def test_category_match_has_lower_confidence(self) -> None:
        """Brand found only in category gets 0.75 confidence."""
        result = self.extractor.extract(
            title="レザーバッグ",
            description="",
            category="PRADA バッグ",
        )

        assert result is not None
        assert result.confidence == 0.75

    def test_case_insensitive_matching(self) -> None:
        """Brand names match regardless of case."""
        result = self.extractor.extract(
            title="chanel bag",
            description="",
            category="",
        )

        assert result is not None
        assert result.brand_name == "CHANEL"

    def test_japanese_brand_name_extraction(self) -> None:
        """Japanese brand names are also extracted."""
        result = self.extractor.extract(
            title="グッチ バッグ 新品",
            description="",
            category="",
        )

        assert result is not None
        assert result.brand_name == "グッチ"

    def test_louis_vuitton_multi_word_brand(self) -> None:
        """Multi-word brand names like LOUIS VUITTON are matched."""
        result = self.extractor.extract(
            title="LOUIS VUITTON モノグラム",
            description="",
            category="",
        )

        assert result is not None
        assert result.brand_name == "LOUIS VUITTON"

    def test_title_takes_priority_over_description(self) -> None:
        """When brand is in both title and description, title wins (0.95)."""
        result = self.extractor.extract(
            title="DIOR サドルバッグ",
            description="GUCCI 風のデザイン",
            category="",
        )

        assert result is not None
        assert result.confidence == 0.95

    def test_no_false_positive_on_partial_match(self) -> None:
        """Partial word matches should not trigger for ASCII brands."""
        result = self.extractor.extract(
            title="coaching session bag",
            description="",
            category="",
        )

        # "coaching" should NOT match "COACH" due to word boundary
        assert result is None


class TestExtractionResult:
    """Tests for the ExtractionResult dataclass."""

    def test_dataclass_fields(self) -> None:
        """ExtractionResult has the expected fields."""
        result = ExtractionResult(brand_name="CHANEL", confidence=0.95)
        assert result.brand_name == "CHANEL"
        assert result.confidence == 0.95


class TestKnownBrands:
    """Tests for the KNOWN_BRANDS constant."""

    def test_known_brands_not_empty(self) -> None:
        """KNOWN_BRANDS list is populated."""
        assert len(KNOWN_BRANDS) > 0

    def test_known_brands_contains_expected_entries(self) -> None:
        """KNOWN_BRANDS contains key brand names."""
        assert "CHANEL" in KNOWN_BRANDS
        assert "シャネル" in KNOWN_BRANDS
        assert "GUCCI" in KNOWN_BRANDS
        assert "グッチ" in KNOWN_BRANDS
