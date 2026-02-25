from __future__ import annotations

import re
from dataclasses import dataclass
from typing import List, Optional, Tuple


@dataclass
class ExtractionResult:
    """Result of a brand extraction operation."""
    brand_name: str
    confidence: float


# Well-known brand names in both English and Japanese
KNOWN_BRANDS = [
    "CHANEL", "シャネル",
    "GUCCI", "グッチ",
    "LOUIS VUITTON", "ルイ・ヴィトン",
    "PRADA", "プラダ",
    "HERMES", "エルメス",
    "BURBERRY", "バーバリー",
    "COACH", "コーチ",
    "NIKE", "ナイキ",
    "ADIDAS", "アディダス",
    "SUPREME", "シュプリーム",
    "BALENCIAGA", "バレンシアガ",
    "DIOR", "ディオール",
    "FENDI", "フェンディ",
    "CELINE", "セリーヌ",
]  # type: List[str]


class BrandExtractor:
    """Extracts brand names from product listing text using pattern matching.

    Searches for known brand names in the title, description, and category
    of a product listing. Confidence is higher (0.95) when the brand is
    found in the title, and lower (0.75) when found only in the description
    or category.
    """

    def __init__(self) -> None:
        """Build regex patterns from KNOWN_BRANDS list.

        Patterns are compiled with case-insensitive matching so that
        "chanel", "CHANEL", and "Chanel" all match. Japanese brand names
        are matched as-is (case insensitivity has no effect on them, but
        causes no harm either).

        Brands are sorted longest-first so that "LOUIS VUITTON" is matched
        before a hypothetical shorter substring.
        """
        sorted_brands = sorted(KNOWN_BRANDS, key=len, reverse=True)
        self._patterns = []  # type: List[Tuple[re.Pattern, str]]
        for brand in sorted_brands:
            # Use word boundary for ASCII brands; for Japanese brands
            # (which contain non-ASCII characters), just look for the
            # substring since \b does not work well with CJK.
            if brand.isascii():
                pattern = re.compile(
                    r'\b' + re.escape(brand) + r'\b',
                    re.IGNORECASE,
                )
            else:
                pattern = re.compile(re.escape(brand))
            self._patterns.append((pattern, brand))

    def extract(
        self,
        title: str,
        description: str = "",
        category: str = "",
    ) -> Optional[ExtractionResult]:
        """Extract a brand name from a product listing.

        Searches the title first (confidence 0.95), then description and
        category (confidence 0.75). Returns the first match found, or
        None if no known brand is detected.

        Args:
            title: The product title/name.
            description: The product description.
            category: The product category.

        Returns:
            An ExtractionResult if a known brand is found, otherwise None.
        """
        # Search in title first (highest confidence)
        for pattern, brand_name in self._patterns:
            if pattern.search(title):
                return ExtractionResult(
                    brand_name=brand_name,
                    confidence=0.95,
                )

        # Search in description and category (lower confidence)
        combined_other = description + " " + category
        for pattern, brand_name in self._patterns:
            if pattern.search(combined_other):
                return ExtractionResult(
                    brand_name=brand_name,
                    confidence=0.75,
                )

        return None
