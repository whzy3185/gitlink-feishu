package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
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
			if envelope.Error.Code != nil && fmt.Sprintf("%v", envelope.Error.Code) != "" {
				fmt.Fprintf(w, "Error [%v]: %s\n", envelope.Error.Code, envelope.Error.Message)
			} else {
				fmt.Fprintf(w, "Error: %s\n", envelope.Error.Message)
			}
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

	// Try to render as table if data is a slice of maps
	switch data := envelope.Data.(type) {
	case []interface{}:
		return printSliceTable(w, data)
	case map[string]interface{}:
		// Resource-wrapped list responses ({"total_count": N, "<resource>": [...]})
		// render the wrapped array as a table with the scalar fields as a summary.
		if items, ok := unwrapListMap(w, data); ok {
			return printSliceTable(w, items)
		}
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

// unwrapListMap detects a map containing exactly one array value while every
// other value is a scalar (the shape of the platform's paginated list
// responses). It prints the scalar fields as a summary line and returns the
// wrapped array for table rendering.
func unwrapListMap(w io.Writer, m map[string]interface{}) ([]interface{}, bool) {
	var items []interface{}
	arrays := 0
	for _, v := range m {
		switch value := v.(type) {
		case []interface{}:
			arrays++
			items = value
		case map[string]interface{}:
			return nil, false
		}
	}
	if arrays != 1 {
		return nil, false
	}
	scalars := make([]string, 0, len(m))
	for _, k := range sortedKeys(m) {
		if _, ok := m[k].([]interface{}); ok {
			continue
		}
		scalars = append(scalars, fmt.Sprintf("%s: %s", k, formatValue(m[k])))
	}
	if len(scalars) > 0 {
		fmt.Fprintln(w, strings.Join(scalars, "  "))
	}
	return items, true
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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
	// collectKeys yields a deterministic order (priority keys first, then the
	// remaining keys sorted), so table output is stable across runs.
	for _, k := range collectKeys(m) {
		fmt.Fprintf(tw, "%s\t%s\n", k, formatValue(m[k]))
	}
	return tw.Flush()
}

func collectKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	// Prefer common keys first
	priority := []string{"number", "id", "name", "login", "title", "status", "state", "created_at", "updated_at"}
	seen := map[string]bool{}
	for _, k := range priority {
		if _, ok := m[k]; ok {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	remaining := make([]string, 0, len(m))
	for k := range m {
		if !seen[k] {
			remaining = append(remaining, k)
		}
	}
	// Sort the non-priority keys so column/row order is deterministic instead of
	// depending on Go's randomized map iteration order.
	sort.Strings(remaining)
	keys = append(keys, remaining...)
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
