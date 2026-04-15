package rss

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)


// Recipe defines site-specific extraction rules (FiveFilters compatible).
type Recipe struct {
	Body   []string
	Strip  []string
	Title  string
	Source string // Path or origin
}


// FindRecipe searches for a config file matching the URL's domain.
func FindRecipe(targetURL string) (*Recipe, error) {
	u, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	host := strings.ToLower(u.Hostname())
	if host == "" {
		return nil, fmt.Errorf("invalid hostname")
	}

	// Determine recipes path (support running from root or internal/rss)
	basePath := "data/site-config/ftr-site-config"
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		basePath = "../../data/site-config/ftr-site-config"
	}

	// Try custom recipes first, then standard repository
	customPath := filepath.Join(filepath.Dir(basePath), "custom")
	
	parts := strings.Split(host, ".")
	// Check custom first
	for i := 0; i < len(parts)-1; i++ {
		domain := strings.Join(parts[i:], ".")
		path := filepath.Join(customPath, domain+".txt")
		if _, err := os.Stat(path); err == nil {
			recipe, err := ParseRecipe(path)
			if err == nil {
				recipe.Source = "custom"
				debugLog("DEBUG: Found custom recipe at %s\n", path)
				return recipe, nil
			}
		}
	}
	// Then check standard
	for i := 0; i < len(parts)-1; i++ {
		domain := strings.Join(parts[i:], ".")
		path := filepath.Join(basePath, domain+".txt")
		if _, err := os.Stat(path); err == nil {
			recipe, err := ParseRecipe(path)
			if err == nil {
				recipe.Source = "system"
				debugLog("DEBUG: Found recipe at %s\n", path)
				return recipe, nil
			}
		}
	}

	return nil, fmt.Errorf("no recipe found for %s", host)
}

// ParseRecipe reads a .txt config file and extracts rules.
func ParseRecipe(path string) (*Recipe, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	recipe := &Recipe{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "body":
			recipe.Body = append(recipe.Body, val)
		case "strip":
			recipe.Strip = append(recipe.Strip, val)
		case "title":
			recipe.Title = val
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return recipe, nil
}

// ApplyRecipe applies the rules to an HTML document.
func ApplyRecipe(root *html.Node, recipe *Recipe) (string, error) {
	if recipe == nil {
		return "", fmt.Errorf("nil recipe")
	}

	debugLog("DEBUG: Applying recipe with %d body rules and %d strip rules\n", len(recipe.Body), len(recipe.Strip))
	var results []string
	var processedNodes []*html.Node

	// 2. Extract Body
	if len(recipe.Body) > 0 {
		for i, xpath := range recipe.Body {
			nodes, err := htmlquery.QueryAll(root, xpath)
			if err != nil {
				fmt.Printf("DEBUG: Body XPath [%d] (%s) error: %v\n", i, xpath, err)
				continue
			}
			
			debugLog("DEBUG: Body XPath [%d] (%s) found %d nodes\n", i, xpath, len(nodes))
			for j, node := range nodes {
				isNested := false
				for _, p := range processedNodes {
					if isAncestor(p, node) {
						isNested = true
						break
					}
				}
				if isNested {
					debugLog("DEBUG: Node [%d.%d] is nested, skipping\n", i, j)
					continue
				}

				// Apply Strip noise INSIDE the body node BEFORE outputting
				for k, stripXpath := range recipe.Strip {
					toStrip, _ := htmlquery.QueryAll(node, stripXpath)
					if len(toStrip) > 0 {
						debugLog("DEBUG: Strip rule [%d] (%s) removed %d elements from node [%d.%d]\n", k, stripXpath, len(toStrip), i, j)
					}
					for _, s := range toStrip {
						if s.Parent != nil {
							s.Parent.RemoveChild(s)
						}
					}
				}

				htmlContent := htmlquery.OutputHTML(node, true)
				// Simple deduplication by content
				alreadyAdded := false
				for _, r := range results {
					if r == htmlContent {
						alreadyAdded = true
						break
					}
				}

				if !alreadyAdded {
					results = append(results, htmlContent)
					processedNodes = append(processedNodes, node)
					debugLog("DEBUG: Added content from node [%d.%d] (size: %d)\n", i, j, len(htmlContent))
				} else {
					debugLog("DEBUG: Node [%d.%d] has identical content to previous, skipping\n", i, j)
				}
			}
		}
	}

	if len(results) == 0 {
		return "", fmt.Errorf("recipe applied but no content found")
	}

	return strings.Join(results, "\n"), nil
}

func isAncestor(ancestor, node *html.Node) bool {
	for n := node.Parent; n != nil; n = n.Parent {
		if n == ancestor {
			return true
		}
	}
	return false
}
