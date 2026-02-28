from __future__ import annotations

from typing import Optional

from fastapi import FastAPI
from pydantic import BaseModel

from app.brand_extractor import BrandExtractor
from app.translation import TranslationService

app = FastAPI(title="Rakutao AI Service", version="0.1.0")

# Singletons
_translation_service = TranslationService(
    brand_mappings={
        "gucci": "グッチ",
        "chanel": "シャネル",
        "louis vuitton": "ルイ・ヴィトン",
        "prada": "プラダ",
        "hermes": "エルメス",
        "burberry": "バーバリー",
        "coach": "コーチ",
        "nike": "ナイキ",
        "adidas": "アディダス",
        "supreme": "シュプリーム",
        "balenciaga": "バレンシアガ",
        "dior": "ディオール",
        "fendi": "フェンディ",
        "celine": "セリーヌ",
    }
)
_brand_extractor = BrandExtractor()


# --- Request/Response models ---


class TranslateRequest(BaseModel):
    keyword: str
    source_lang: Optional[str] = None


class TranslateResponse(BaseModel):
    keyword_ja: str
    original: str
    source_lang: str
    was_translated: bool


class ExtractBrandRequest(BaseModel):
    title: str
    description: str = ""
    category: str = ""


class ExtractBrandResponse(BaseModel):
    brand_name: Optional[str] = None
    confidence: float = 0.0


class HealthResponse(BaseModel):
    status: str


# --- Endpoints ---


@app.post("/translate", response_model=TranslateResponse)
def translate(req: TranslateRequest) -> TranslateResponse:
    result = _translation_service.translate_keyword(
        keyword=req.keyword,
        source_lang=req.source_lang,
    )
    return TranslateResponse(
        keyword_ja=result.keyword_ja,
        original=result.original,
        source_lang=result.source_lang,
        was_translated=result.was_translated,
    )


@app.post("/extract-brand", response_model=ExtractBrandResponse)
def extract_brand(req: ExtractBrandRequest) -> ExtractBrandResponse:
    result = _brand_extractor.extract(
        title=req.title,
        description=req.description,
        category=req.category,
    )
    if result is None:
        return ExtractBrandResponse(brand_name=None, confidence=0.0)
    return ExtractBrandResponse(
        brand_name=result.brand_name,
        confidence=result.confidence,
    )


@app.get("/health", response_model=HealthResponse)
def health() -> HealthResponse:
    return HealthResponse(status="healthy")
