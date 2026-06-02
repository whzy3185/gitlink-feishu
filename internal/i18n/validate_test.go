package i18n

import "testing"

func TestValidateEmbeddedLocales(t *testing.T) {
	problems, err := Validate(NewEmbedLoader(), "en-US")
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Fatalf("expected no problems, got %v", problems)
	}
}

func TestValidateFindsMissingKeyAndArgMismatch(t *testing.T) {
	problems, err := Validate(mapLoader{messages: map[string]map[string]string{
		"en-US": {
			"cmd.root.short": "Hello {name}",
			"flag.owner":     "Owner",
		},
		"zh-CN": {
			"cmd.root.short": "你好",
		},
	}}, "en-US")
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 2 {
		t.Fatalf("expected 2 problems, got %d: %v", len(problems), problems)
	}
}
