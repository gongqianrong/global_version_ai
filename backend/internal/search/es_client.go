// Package search implements Elasticsearch query building and the search
// gateway that orchestrates keyword translation, query construction, and
// result aggregation for the Rakutao collection gateway.
package search

import (
	"encoding/json"

	"github.com/rakutao/collection-gateway/internal/domain"
)

const defaultPageSize = 20

// buildESQuery constructs an Elasticsearch query JSON body from the given
// domain.SearchQuery. It builds a bool query with multi_match on text fields,
// applies filters for platform/brand/price/condition/categories/status, adds
// sort clauses, aggregations, and pagination. The returned bytes are the
// JSON-encoded query ready to be sent to the ES _search endpoint.
func buildESQuery(q domain.SearchQuery) []byte {
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	page := q.Page
	if page <= 0 {
		page = 1
	}
	from := (page - 1) * pageSize

	// --- bool query ---
	boolQuery := map[string]interface{}{}

	// must clause: multi_match or match_all
	if q.KeywordJA != "" {
		boolQuery["must"] = []interface{}{
			map[string]interface{}{
				"multi_match": map[string]interface{}{
					"query":  q.KeywordJA,
					"fields": []string{"title^3", "title_en^2", "title_zh_tw^2", "description", "brand_name_ja"},
					"type":   "best_fields",
				},
			},
		}
	} else {
		boolQuery["must"] = []interface{}{
			map[string]interface{}{
				"match_all": map[string]interface{}{},
			},
		}
	}

	// filter clauses
	filters := []interface{}{
		// Always filter for available status
		map[string]interface{}{
			"term": map[string]interface{}{
				"status": "available",
			},
		},
	}

	if len(q.Platforms) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{
				"source_platform": q.Platforms,
			},
		})
	}

	if q.BrandID != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"brand_id": q.BrandID,
			},
		})
	}

	if q.PriceMin > 0 || q.PriceMax > 0 {
		rangeFilter := map[string]interface{}{}
		if q.PriceMin > 0 {
			rangeFilter["gte"] = q.PriceMin
		}
		if q.PriceMax > 0 {
			rangeFilter["lte"] = q.PriceMax
		}
		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{
				"price_jpy": rangeFilter,
			},
		})
	}

	if len(q.Condition) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{
				"condition": q.Condition,
			},
		})
	}

	if len(q.Categories) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{
				"categories": q.Categories,
			},
		})
	}

	boolQuery["filter"] = filters

	// --- sort ---
	sort := buildSort(q.SortBy)

	// --- aggregations ---
	aggs := map[string]interface{}{
		"platforms": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "source_platform",
				"size":  50,
			},
		},
		"brands": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "brand_name",
				"size":  50,
			},
		},
		"categories": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "categories",
				"size":  50,
			},
		},
		"price_stats": map[string]interface{}{
			"stats": map[string]interface{}{
				"field": "price_jpy",
			},
		},
	}

	// --- assemble full query ---
	body := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
		"sort": sort,
		"aggs": aggs,
		"from": from,
		"size": pageSize,
	}

	data, _ := json.Marshal(body)
	return data
}

// buildSort returns the ES sort clause for the given sort key.
func buildSort(sortBy string) []interface{} {
	switch sortBy {
	case "price_asc":
		return []interface{}{
			map[string]interface{}{
				"price_jpy": map[string]interface{}{"order": "asc"},
			},
		}
	case "price_desc":
		return []interface{}{
			map[string]interface{}{
				"price_jpy": map[string]interface{}{"order": "desc"},
			},
		}
	case "newest":
		return []interface{}{
			map[string]interface{}{
				"listed_at": map[string]interface{}{"order": "desc"},
			},
		}
	default:
		return []interface{}{
			map[string]interface{}{
				"_score": map[string]interface{}{"order": "desc"},
			},
		}
	}
}
