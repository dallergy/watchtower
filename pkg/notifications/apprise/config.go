package apprise

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

// LoadConfig reads the notification URLs from an Apprise configuration file.
//
// Both the TEXT format (one URL per line, optionally prefixed with "tags=") and the YAML format
// (a "urls" list) are supported. Per-URL options and tags in YAML files are ignored.
func LoadConfig(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read apprise config: %w", err)
	}

	if isYAMLConfig(path, content) {
		urls, err := parseYAMLConfig(content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse apprise config %s: %w", path, err)
		}
		return urls, nil
	}
	return parseTextConfig(content), nil
}

var yamlURLsKey = regexp.MustCompile(`(?m)^urls\s*:`)

func isYAMLConfig(path string, content []byte) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml") || yamlURLsKey.Match(content)
}

func parseTextConfig(content []byte) []string {
	var urls []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		// Strip the optional "tag1, tag2=" prefix, taking care not to cut inside the URL itself
		if eq := strings.Index(line, "="); eq >= 0 && !strings.Contains(line[:eq], "://") {
			line = strings.TrimSpace(line[eq+1:])
		}
		if line != "" {
			urls = append(urls, line)
		}
	}
	return urls
}

func parseYAMLConfig(content []byte) ([]string, error) {
	var config struct {
		URLs []yaml.Node `yaml:"urls"`
	}
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, err
	}

	var urls []string
	for _, entry := range config.URLs {
		switch entry.Kind {
		case yaml.ScalarNode:
			// - gotifys://host/token
			urls = append(urls, entry.Value)
		case yaml.MappingNode:
			// - gotifys://host/token:
			//     tag: foo
			for i := 0; i < len(entry.Content); i += 2 {
				urls = append(urls, entry.Content[i].Value)
			}
		default:
			return nil, fmt.Errorf("unexpected entry on line %d", entry.Line)
		}
	}
	return urls, nil
}
