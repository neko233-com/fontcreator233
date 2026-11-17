package recipe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads and validates a recipe from disk.
func Load(path string) (*Recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	r, err := Parse(data, path)
	if err != nil {
		return nil, err
	}
	r.SourcePath = path
	return r, nil
}

// Parse decodes YAML (default) or JSON (when filename ends in .json), merges
// defaults, normalizes, and validates.
func Parse(data []byte, filename string) (*Recipe, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("empty recipe")
	}
	r := Default()
	var err error
	if strings.EqualFold(filepath.Ext(filename), ".json") {
		err = parseJSON(data, r)
	} else {
		err = parseYAML(data, r)
	}
	if err != nil {
		return nil, err
	}
	if filename != "" {
		r.SourcePath = filename
	}
	r.Normalize()
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

func parseJSON(data []byte, r *Recipe) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(r); err != nil {
		return fmt.Errorf("decode JSON recipe: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode JSON recipe: trailing data")
		}
		return fmt.Errorf("decode JSON recipe: %w", err)
	}
	return nil
}

func parseYAML(data []byte, r *Recipe) error {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("decode YAML recipe: %w", err)
	}
	if root.Kind == 0 {
		return fmt.Errorf("empty recipe")
	}
	if err := checkDuplicateKeys(&root, ""); err != nil {
		return err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(r); err != nil {
		return fmt.Errorf("decode YAML recipe: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode YAML recipe: trailing document")
		}
		return fmt.Errorf("decode YAML recipe: %w", err)
	}
	return nil
}

func checkDuplicateKeys(n *yaml.Node, path string) error {
	if n == nil {
		return nil
	}
	switch n.Kind {
	case yaml.DocumentNode:
		for _, c := range n.Content {
			if err := checkDuplicateKeys(c, path); err != nil {
				return err
			}
		}
	case yaml.MappingNode:
		seen := map[string]int{}
		for i := 0; i+1 < len(n.Content); i += 2 {
			key := n.Content[i]
			val := n.Content[i+1]
			if key.Kind == yaml.ScalarNode {
				if prev, ok := seen[key.Value]; ok {
					return fmt.Errorf("duplicate key %q (line %d, previously line %d)", key.Value, key.Line, prev)
				}
				seen[key.Value] = key.Line
				next := key.Value
				if path != "" {
					next = path + "." + key.Value
				}
				if err := checkDuplicateKeys(val, next); err != nil {
					return err
				}
			}
		}
	case yaml.SequenceNode:
		for _, c := range n.Content {
			if err := checkDuplicateKeys(c, path+"[]"); err != nil {
				return err
			}
		}
	case yaml.AliasNode:
		// Alias content is checked where it is defined.
	}
	return nil
}
