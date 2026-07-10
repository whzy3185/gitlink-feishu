package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

func Print(envelope *Envelope, format string) error {
	if format == "" {
		format = "json"
	}
	return PrintTo(os.Stdout, envelope, format)
}

func PrintTo(w io.Writer, envelope *Envelope, format string) error {
	switch format {
	case "json":
		return printJSON(w, envelope)
	case "yaml":
		return printYAML(w, envelope)
	case "table":
		return printTable(w, envelope)
	default:
		return printJSON(w, envelope)
	}
}

func printJSON(w io.Writer, envelope *Envelope) error {
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

func printYAML(w io.Writer, envelope *Envelope) error {
	data, err := yaml.Marshal(envelope)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, string(data))
	return err
}

func printTable(w io.Writer, envelope *Envelope) error {
	if !envelope.OK {
		if envelope.Error != nil {
			fmt.Fprintf(w, "Error: %s\n", envelope.Error.Message)
			if envelope.Error.Suggestion != "" {
				fmt.Fprintf(w, "Suggestion: %s\n", envelope.Error.Suggestion)
			}
		}
		return nil
	}

	if envelope.Data == nil {
		fmt.Fprintln(w, "No data")
		return nil
	}

	// Detect and render diff data in git-diff style
	if isDiffData(envelope.Data) {
		return printDiffTable(w, envelope)
	}

	// Try to render as table if data is a slice of maps
	switch data := envelope.Data.(type) {
	case []interface{}:
		return printSliceTable(w, data)
	case map[string]interface{}:
		// For maps with nested structures, prefer JSON
		if hasComplexValues(data) {
			return printJSON(w, envelope)
		}
		return printMapTable(w, data)
	default:
		// Fallback to JSON
		return printJSON(w, envelope)
	}
}

func hasComplexValues(m map[string]interface{}) bool {
	for _, v := range m {
		switch v.(type) {
		case map[string]interface{}, []interface{}:
			return true
		}
	}
	return false
}

func printSliceTable(w io.Writer, items []interface{}) error {
	if len(items) == 0 {
		fmt.Fprintln(w, "No results")
		return nil
	}

	// Collect headers from first item
	first, ok := items[0].(map[string]interface{})
	if !ok {
		data, _ := json.MarshalIndent(items, "", "  ")
		fmt.Fprintln(w, string(data))
		return nil
	}

	headers := collectKeys(first)
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	// Print headers
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	dashes := make([]string, len(headers))
	for i, h := range headers {
		dashes[i] = strings.Repeat("-", len(h))
	}
	fmt.Fprintln(tw, strings.Join(dashes, "\t"))

	// Print rows
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		vals := make([]string, len(headers))
		for i, h := range headers {
			vals[i] = formatValue(m[h])
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}
	return tw.Flush()
}

func printMapTable(w io.Writer, m map[string]interface{}) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "KEY\tVALUE")
	fmt.Fprintln(tw, "---\t-----")
	for k, v := range m {
		fmt.Fprintf(tw, "%s\t%s\n", k, formatValue(v))
	}
	return tw.Flush()
}

func collectKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	// Prefer common keys first
	priority := []string{"id", "name", "login", "title", "status", "state", "created_at", "updated_at"}
	seen := map[string]bool{}
	for _, k := range priority {
		if _, ok := m[k]; ok {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	for k := range m {
		if !seen[k] {
			keys = append(keys, k)
		}
	}
	return keys
}

func formatValue(v interface{}) string {
	if v == nil {
		return ""
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map, reflect.Slice:
		data, _ := json.Marshal(v)
		s := string(data)
		if len(s) > 60 {
			return s[:57] + "..."
		}
		return s
	default:
		return fmt.Sprintf("%v", v)
	}
}

// isDiffData checks whether the envelope data is a PR diff response with sections.
// Distinguishes from the simpler files listing by checking for sections in files.
func isDiffData(data interface{}) bool {
	m, ok := data.(map[string]interface{})
	if !ok {
		return false
	}
	files, hasFiles := m["files"].([]interface{})
	if !hasFiles || len(files) == 0 {
		return false
	}
	// Diff data has files with "sections"; simple file listing does not.
	firstFile, ok := files[0].(map[string]interface{})
	if !ok {
		return false
	}
	_, hasSections := firstFile["sections"]
	return hasSections
}

// printDiffTable renders diff data in git-diff style text output.
func printDiffTable(w io.Writer, envelope *Envelope) error {
	data, ok := envelope.Data.(map[string]interface{})
	if !ok {
		return printJSON(w, envelope)
	}

	files, ok := data["files"].([]interface{})
	if !ok {
		return printJSON(w, envelope)
	}

	// Summary header
	fileNums, _ := data["file_nums"].(float64)
	totalAdd, _ := data["total_addition"].(float64)
	totalDel, _ := data["total_deletion"].(float64)
	fmt.Fprintf(w, " %d files changed, %d insertions(+), %d deletions(-)\n\n",
		int(fileNums), int(totalAdd), int(totalDel))

	for _, f := range files {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := fm["name"].(string)
		addition, _ := fm["addition"].(float64)
		deletion, _ := fm["deletion"].(float64)

		// File header
		fmt.Fprintf(w, "diff --git a/%s b/%s\n", name, name)

		if isCreated, _ := fm["is_created"].(bool); isCreated {
			fmt.Fprintf(w, "new file\n")
		}
		if isDeleted, _ := fm["is_deleted"].(bool); isDeleted {
			fmt.Fprintf(w, "deleted file\n")
		}

		fmt.Fprintf(w, "--- a/%s\n", name)
		fmt.Fprintf(w, "+++ b/%s\n", name)
		fmt.Fprintf(w, "@@ +%d -%d @@\n", int(addition), int(deletion))

		// Render each line
		sections, _ := fm["sections"].([]interface{})
		for _, sec := range sections {
			secMap, ok := sec.(map[string]interface{})
			if !ok {
				continue
			}
			lines, _ := secMap["lines"].([]interface{})
			for _, l := range lines {
				lineMap, ok := l.(map[string]interface{})
				if !ok {
					continue
				}
				content, _ := lineMap["content"].(string)
				lineType, _ := lineMap["type"].(float64)

				switch int(lineType) {
				case 4: // diff hunk header
					fmt.Fprintf(w, "%s\n", content)
				case 2: // addition
					fmt.Fprintf(w, "%s\n", content)
				case 3: // deletion
					fmt.Fprintf(w, "%s\n", content)
				default: // context line
					fmt.Fprintf(w, "%s\n", content)
				}
			}
		}
		fmt.Fprintln(w)
	}
	return nil
}
