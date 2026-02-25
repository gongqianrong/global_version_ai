package brand

import (
	"testing"
	"time"
)

func newTestRegistry() *Registry {
	r := NewRegistry()

	r.Add(Entry{
		ID:         "louis_vuitton",
		NameStd:    "Louis Vuitton",
		NameJA:     "ルイ・ヴィトン",
		NameZhTW:   "路易威登",
		Aliases:    []string{"LV", "ルイヴィトン", "LOUIS VUITTON"},
		Category:   "luxury",
		IsVerified: true,
		CreatedAt:  time.Now(),
	})

	r.Add(Entry{
		ID:         "gucci",
		NameStd:    "Gucci",
		NameJA:     "グッチ",
		NameZhTW:   "古馳",
		Aliases:    []string{"GUCCI", "グッチ"},
		Category:   "luxury",
		IsVerified: true,
		CreatedAt:  time.Now(),
	})

	r.Add(Entry{
		ID:         "chanel",
		NameStd:    "Chanel",
		NameJA:     "シャネル",
		NameZhTW:   "香奈兒",
		Aliases:    []string{"CHANEL", "シャネル"},
		Category:   "luxury",
		IsVerified: true,
		CreatedAt:  time.Now(),
	})

	return r
}

// --- Test 1: FindByAlias("LV") returns Louis Vuitton entry ---

func TestFindByAlias_LV(t *testing.T) {
	r := newTestRegistry()

	entry := r.FindByAlias("LV")
	if entry == nil {
		t.Fatal("expected entry for 'LV', got nil")
	}
	if entry.ID != "louis_vuitton" {
		t.Errorf("ID = %q, want %q", entry.ID, "louis_vuitton")
	}
	if entry.NameStd != "Louis Vuitton" {
		t.Errorf("NameStd = %q, want %q", entry.NameStd, "Louis Vuitton")
	}
}

// --- Test: FindByAlias is case-insensitive for Latin ---

func TestFindByAlias_CaseInsensitive(t *testing.T) {
	r := newTestRegistry()

	tests := []struct {
		alias string
		wantID string
	}{
		{"lv", "louis_vuitton"},
		{"Lv", "louis_vuitton"},
		{"gucci", "gucci"},
		{"Gucci", "gucci"},
		{"louis vuitton", "louis_vuitton"},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			entry := r.FindByAlias(tt.alias)
			if entry == nil {
				t.Fatalf("expected entry for %q, got nil", tt.alias)
			}
			if entry.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", entry.ID, tt.wantID)
			}
		})
	}
}

// --- Test 2: FindByAlias("unknown") returns nil ---

func TestFindByAlias_Unknown(t *testing.T) {
	r := newTestRegistry()

	entry := r.FindByAlias("unknown")
	if entry != nil {
		t.Errorf("expected nil for 'unknown', got %+v", entry)
	}
}

// --- Test 3: MatchInText("グッチ レザー バッグ") returns Gucci ---

func TestMatchInText_Gucci(t *testing.T) {
	r := newTestRegistry()

	entry := r.MatchInText("グッチ レザー バッグ")
	if entry == nil {
		t.Fatal("expected Gucci entry, got nil")
	}
	if entry.ID != "gucci" {
		t.Errorf("ID = %q, want %q", entry.ID, "gucci")
	}
}

// --- Test 4: MatchInText("ノーブランド バッグ") returns nil ---

func TestMatchInText_NoBrand(t *testing.T) {
	r := newTestRegistry()

	entry := r.MatchInText("ノーブランド バッグ")
	if entry != nil {
		t.Errorf("expected nil for 'ノーブランド バッグ', got %+v", entry)
	}
}

// --- Test: MatchInText finds brand by standard name ---

func TestMatchInText_StandardName(t *testing.T) {
	r := newTestRegistry()

	entry := r.MatchInText("Beautiful Chanel handbag for sale")
	if entry == nil {
		t.Fatal("expected Chanel entry, got nil")
	}
	if entry.ID != "chanel" {
		t.Errorf("ID = %q, want %q", entry.ID, "chanel")
	}
}

// --- Test: All() returns all entries ---

func TestAll(t *testing.T) {
	r := newTestRegistry()

	all := r.All()
	if len(all) != 3 {
		t.Errorf("All() returned %d entries, want 3", len(all))
	}
}

// --- Test: FindByAlias with Japanese alias ---

func TestFindByAlias_Japanese(t *testing.T) {
	r := newTestRegistry()

	entry := r.FindByAlias("ルイ・ヴィトン")
	if entry == nil {
		t.Fatal("expected entry for 'ルイ・ヴィトン', got nil")
	}
	if entry.ID != "louis_vuitton" {
		t.Errorf("ID = %q, want %q", entry.ID, "louis_vuitton")
	}
}
