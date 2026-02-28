from __future__ import annotations

from starlette.testclient import TestClient

from app.main import app

client = TestClient(app)


class TestTranslateEndpoint:
    def test_translate_english_keyword(self) -> None:
        resp = client.post("/translate", json={
            "keyword": "gucci bag",
            "source_lang": "en",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["original"] == "gucci bag"
        assert data["source_lang"] == "en"
        assert data["was_translated"] is True
        assert "グッチ" in data["keyword_ja"]

    def test_translate_japanese_keyword(self) -> None:
        resp = client.post("/translate", json={
            "keyword": "グッチ バッグ",
            "source_lang": "ja",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["was_translated"] is False
        assert data["keyword_ja"] == "グッチ バッグ"

    def test_translate_auto_detect(self) -> None:
        resp = client.post("/translate", json={
            "keyword": "グッチ バッグ",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["source_lang"] == "ja"
        assert data["was_translated"] is False

    def test_translate_missing_keyword(self) -> None:
        resp = client.post("/translate", json={})
        assert resp.status_code == 422


class TestExtractBrandEndpoint:
    def test_extract_brand_from_title(self) -> None:
        resp = client.post("/extract-brand", json={
            "title": "GUCCI バッグ 新品",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["brand_name"] == "GUCCI"
        assert data["confidence"] == 0.95

    def test_extract_brand_from_description(self) -> None:
        resp = client.post("/extract-brand", json={
            "title": "高級バッグ",
            "description": "CHANEL の商品です",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["brand_name"] == "CHANEL"
        assert data["confidence"] == 0.75

    def test_extract_brand_no_match(self) -> None:
        resp = client.post("/extract-brand", json={
            "title": "手作りバッグ",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["brand_name"] is None
        assert data["confidence"] == 0.0


class TestHealthEndpoint:
    def test_health(self) -> None:
        resp = client.get("/health")
        assert resp.status_code == 200
        assert resp.json() == {"status": "healthy"}
