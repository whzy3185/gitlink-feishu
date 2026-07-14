package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Query holds the global --query dot-path; when non-empty, Print extracts
// and prints only the matching value instead of the full envelope.
var Query string

// PrintQuery extracts a value from the envelope by a dot-separated path and
// prints it. Path segments are object keys; non-negative integers index into
// arrays (e.g. "data.commits.0.sha"). Strings and other scalars are printed
// as raw values so results can be consumed directly by shell pipelines;
// objects and arrays are printed as indented JSON.
func PrintQuery(w io.Writer, envelope *Envelope, path string) error {
	raw, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	var root interface{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	value, err := resolvePath(root, path)
	if err != nil {
		return err
	}
	switch v := value.(type) {
	case string:
		_, err = fmt.Fprintln(w, v)
	case nil:
		_, err = fmt.Fprintln(w, "null")
	case float64, bool:
		_, err = fmt.Fprintln(w, v)
	default:
		data, merr := json.MarshalIndent(v, "", "  ")
		if merr != nil {
			return merr
		}
		_, err = fmt.Fprintln(w, string(data))
	}
	return err
}

func resolvePath(root interface{}, path string) (interface{}, error) {
	current := root
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("query path is empty")
	}
	for _, segment := range strings.Split(path, ".") {
		if segment == "" {
			return nil, fmt.Errorf("query path %q contains an empty segment", path)
		}
		switch node := current.(type) {
		case map[string]interface{}:
			value, ok := node[segment]
			if !ok {
				return nil, fmt.Errorf("query key %q not found (available: %s)", segment, strings.Join(mapKeys(node), ", "))
			}
			current = value
		case []interface{}:
			index, err := strconv.Atoi(segment)
			if err != nil {
				return nil, fmt.Errorf("query segment %q must be an array index (array has %d items)", segment, len(node))
			}
			if index < 0 || index >= len(node) {
				return nil, fmt.Errorf("query index %d out of range (array has %d items)", index, len(node))
			}
			current = node[index]
		default:
			return nil, fmt.Errorf("query segment %q cannot descend into a scalar value", segment)
		}
	}
	return current, nil
}

func mapKeys(node map[string]interface{}) []string {
	keys := make([]string, 0, len(node))
	for key := range node {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
