package filter

import "testing"

func TestKeywordFilter_ExactMatch(t *testing.T) {
	f := NewKeywordFilter([]string{"天安门", "六四"}, nil)

	if !f.IsBlocked("天安门") {
		t.Error("expected '天安门' to be blocked (exact match)")
	}
	if !f.IsBlocked("六四") {
		t.Error("expected '六四' to be blocked (exact match)")
	}
	if f.IsBlocked("动漫") {
		t.Error("expected '动漫' NOT to be blocked")
	}
}

func TestKeywordFilter_ContainsMatch(t *testing.T) {
	f := NewKeywordFilter(nil, []string{"政治"})

	if !f.IsBlocked("日本政治ニュース") {
		t.Error("expected '日本政治ニュース' to be blocked (contains '政治')")
	}
	if !f.IsBlocked("政治") {
		t.Error("expected '政治' to be blocked (exact substring)")
	}
	if f.IsBlocked("経済") {
		t.Error("expected '経済' NOT to be blocked")
	}
}

func TestKeywordFilter_CaseInsensitive(t *testing.T) {
	f := NewKeywordFilter([]string{"Blocked"}, []string{"sensitive"})

	if !f.IsBlocked("BLOCKED") {
		t.Error("expected 'BLOCKED' to be blocked (case insensitive exact)")
	}
	if !f.IsBlocked("blocked") {
		t.Error("expected 'blocked' to be blocked (case insensitive exact)")
	}
	if !f.IsBlocked("This is Sensitive content") {
		t.Error("expected 'This is Sensitive content' to be blocked (case insensitive contains)")
	}
}

func TestKeywordFilter_EmptyQuery(t *testing.T) {
	f := NewKeywordFilter([]string{"test"}, []string{"test"})

	if f.IsBlocked("") {
		t.Error("expected empty query NOT to be blocked")
	}
}

func TestKeywordFilter_EmptyFilter(t *testing.T) {
	f := NewKeywordFilter(nil, nil)

	if f.IsBlocked("anything") {
		t.Error("expected nothing to be blocked with empty filter")
	}
}

func TestKeywordFilter_AddExact(t *testing.T) {
	f := NewKeywordFilter(nil, nil)

	if f.IsBlocked("newkeyword") {
		t.Error("expected 'newkeyword' NOT to be blocked before adding")
	}

	f.AddExact("newkeyword")

	if !f.IsBlocked("newkeyword") {
		t.Error("expected 'newkeyword' to be blocked after adding")
	}
}

func TestKeywordFilter_AddContains(t *testing.T) {
	f := NewKeywordFilter(nil, nil)

	f.AddContains("substr")

	if !f.IsBlocked("this has substr in it") {
		t.Error("expected 'this has substr in it' to be blocked after adding contains")
	}
}

func TestKeywordFilter_RemoveExact(t *testing.T) {
	f := NewKeywordFilter([]string{"removeme"}, nil)

	if !f.IsBlocked("removeme") {
		t.Error("expected 'removeme' to be blocked before removal")
	}

	f.RemoveExact("removeme")

	if f.IsBlocked("removeme") {
		t.Error("expected 'removeme' NOT to be blocked after removal")
	}
}

func TestKeywordFilter_BothMatchTypes(t *testing.T) {
	f := NewKeywordFilter([]string{"exact_block"}, []string{"partial"})

	if !f.IsBlocked("exact_block") {
		t.Error("expected 'exact_block' to be blocked (exact)")
	}
	if !f.IsBlocked("some partial match") {
		t.Error("expected 'some partial match' to be blocked (contains)")
	}
	if f.IsBlocked("no match here") {
		t.Error("expected 'no match here' NOT to be blocked")
	}
}

func TestKeywordFilter_UnicodeKeywords(t *testing.T) {
	f := NewKeywordFilter([]string{"法轮功"}, []string{"独立"})

	if !f.IsBlocked("法轮功") {
		t.Error("expected '法轮功' to be blocked (exact)")
	}
	if !f.IsBlocked("台湾独立运动") {
		t.Error("expected '台湾独立运动' to be blocked (contains '独立')")
	}
	if f.IsBlocked("普通商品") {
		t.Error("expected '普通商品' NOT to be blocked")
	}
}
