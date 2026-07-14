package client

import (
	"bytes"
	"strings"
	"testing"
)

func TestProgressReporterLargeFile(t *testing.T) {
	var buf bytes.Buffer
	total := int64(4 << 20)
	p := newProgressReporterTo("uploading", "big.bin", total, &buf)

	chunk := make([]byte, 1<<20)
	for i := 0; i < 4; i++ {
		if _, err := p.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}
	out := buf.String()
	if !strings.Contains(out, "uploading big.bin") {
		t.Fatalf("missing progress prefix: %q", out)
	}
	if !strings.Contains(out, "100%") {
		t.Fatalf("missing 100%% mark: %q", out)
	}
	if !strings.Contains(out, "4.0 MiB / 4.0 MiB") {
		t.Fatalf("missing byte summary: %q", out)
	}
}

func TestProgressReporterSmallFileSilent(t *testing.T) {
	var buf bytes.Buffer
	p := newProgressReporterTo("uploading", "small.txt", 1024, &buf)
	if _, err := p.Write(make([]byte, 1024)); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no progress output for small file, got %q", buf.String())
	}
}

func TestProgressReporterUnknownTotal(t *testing.T) {
	var buf bytes.Buffer
	p := newProgressReporterTo("downloading", "chunked.bin", -1, &buf)
	chunk := make([]byte, 1<<20)
	for i := 0; i < 3; i++ {
		if _, err := p.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}
	p.Close()
	out := buf.String()
	if !strings.Contains(out, "downloading chunked.bin") {
		t.Fatalf("missing progress prefix: %q", out)
	}
	if !strings.Contains(out, "3.0 MiB") {
		t.Fatalf("missing final byte count: %q", out)
	}
}

func TestProgressReporterUnknownTotalSmallSilent(t *testing.T) {
	var buf bytes.Buffer
	p := newProgressReporterTo("downloading", "small.bin", -1, &buf)
	if _, err := p.Write(make([]byte, 1024)); err != nil {
		t.Fatal(err)
	}
	p.Close()
	if buf.Len() != 0 {
		t.Fatalf("expected no progress output for small unknown-total transfer, got %q", buf.String())
	}
}

func TestFormatBytes(t *testing.T) {
	cases := map[int64]string{
		512:     "512 B",
		2 << 10: "2.0 KiB",
		3 << 20: "3.0 MiB",
		5 << 30: "5.0 GiB",
	}
	for in, want := range cases {
		if got := formatBytes(in); got != want {
			t.Fatalf("formatBytes(%d) = %q, want %q", in, got, want)
		}
	}
}
