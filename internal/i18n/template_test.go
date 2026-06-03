package i18n

import "testing"

func TestRenderTemplateKeepsMissingArgs(t *testing.T) {
	got := renderTemplate("Delete {owner}/{repo}", Args{"owner": "alice"})
	want := "Delete alice/{repo}"
	if got != want {
		t.Fatalf("renderTemplate() = %q, want %q", got, want)
	}
}

func TestExtractTemplateArgs(t *testing.T) {
	got := extractTemplateArgs("Delete {owner}/{repo}/{owner}")
	want := []string{"owner", "repo"}
	if len(got) != len(want) {
		t.Fatalf("args length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
