package workflow

import "testing"

func TestNormalizeLang(t *testing.T) {
	if got := normalizeLang(""); got != langEN {
		t.Fatalf("normalizeLang(\"\") = %q, want %q", got, langEN)
	}
	if got := normalizeLang(langZH); got != langZH {
		t.Fatalf("normalizeLang(%q) = %q, want %q", langZH, got, langZH)
	}
	if got := normalizeLang("fr"); got != langEN {
		t.Fatalf("normalizeLang(\"fr\") = %q, want %q", got, langEN)
	}
}

func TestMessageFallback(t *testing.T) {
	if got := message("fr", "rec_maintain"); got == "" || got == "rec_maintain" {
		t.Fatalf("message fallback = %q, want English message", got)
	}
	if got := message(langEN, "not_found_key"); got != "not_found_key" {
		t.Fatalf("message unknown key = %q, want key", got)
	}
}
