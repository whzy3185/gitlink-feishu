// Command check validates the SKILL.md frontmatter of the skills/ registry.
// It mirrors the i18n gate: run from the repo root, it exits non-zero and
// prints every problem, so CI and `make check` can keep the registry honest.
package main

import (
	"fmt"
	"os"

	"github.com/gitlink-org/gitlink-cli/internal/skillmeta"
)

func main() {
	problems, err := skillmeta.Validate("skills")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, p.String())
		}
		os.Exit(1)
	}
	fmt.Println("skill metadata is valid")
}
