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
			if envelope.Error.Code != nil {
				fmt.Fprintf(w, "%s %s\n", red(fmt.Sprintf("Error [%v]:", envelope.Error.Code)), envelope.Error.Message)
			} else {
				fmt.Fprintf(w, "%s %s\n", red("Error:"), envelope.Error.Message)
			}
			if envelope.Error.Suggestion != "" {
				fmt.Fprintf(w, "%s %s\n", yellow("Suggestion:"), envelope.Error.Suggestion)
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
		if key, items, ok := resourceWrappedList(data); ok {
			for _, summaryKey := range collectKeys(data) {
				if summaryKey != key {
					fmt.Fprintf(w, "%s: %s\n", summaryKey, formatValue(data[summaryKey]))
				}
			}
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

	// Print headers (bold/colored)
	coloredHeaders := make([]string, len(headers))
	for i, h := range headers {
		coloredHeaders[i] = bold(h)
	}
	fmt.Fprintln(tw, strings.Join(coloredHeaders, "\t"))
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
			raw := formatValue(m[h])
			vals[i] = colorForKey(h, raw)
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}
	return tw.Flush()
}

func printMapTable(w io.Writer, m map[string]interface{}) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "KEY\tVALUE")
	fmt.Fprintln(tw, "---\t-----")
	for _, k := range collectKeys(m) {
		v := m[k]
		fmt.Fprintf(tw, "%s\t%s\n", k, formatValue(v))
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
	remaining := make([]string, 0, len(m)-len(keys))
	for k := range m {
		if !seen[k] {
			remaining = append(remaining, k)
		}
	}
	sort.Strings(remaining)
	keys = append(keys, remaining...)
	return keys
}

func resourceWrappedList(data map[string]interface{}) (string, []interface{}, bool) {
	var listKey string
	var items []interface{}
	for key, value := range data {
		if _, nested := value.(map[string]interface{}); nested {
			return "", nil, false
		}
		candidate, ok := value.([]interface{})
		if !ok {
			continue
		}
		if listKey != "" {
			return "", nil, false
		}
		for _, item := range candidate {
			if _, ok := item.(map[string]interface{}); !ok {
				return "", nil, false
			}
		}
		listKey, items = key, candidate
	}
	return listKey, items, listKey != ""
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
