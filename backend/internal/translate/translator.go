package translate

import "context"

// Translator translates text between languages.
type Translator interface {
	Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error)
	BatchTranslate(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error)
}

// Noop is a no-op translator that returns the original text. Used as placeholder
// until a real translation API is integrated.
type Noop struct{}

func NewNoop() *Noop { return &Noop{} }

func (n *Noop) Translate(_ context.Context, text, _, _ string) (string, error) {
	return text, nil
}

func (n *Noop) BatchTranslate(_ context.Context, texts []string, _, _ string) ([]string, error) {
	result := make([]string, len(texts))
	copy(result, texts)
	return result, nil
}
