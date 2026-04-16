package rss

import (
	"bufio"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

// UpdateRecipes performs a git pull on the ftr-site-config repository to get the latest rules.
func UpdateRecipes() error {
	basePath := "data/site-config/ftr-site-config"
	// Check if the directory exists and is a git repo
	if _, err := os.Stat(filepath.Join(basePath, ".git")); os.IsNotExist(err) {
		slog.Warn("Recipes repository not found or not a git repo, skipping update", "path", basePath)
		return nil
	}

	slog.Info("Updating extraction recipes from repository...")
	cmd := exec.Command("git", "-C", basePath, "pull", "origin", "master")
	if output, err := cmd.CombinedOutput(); err != nil {
		slog.Error("Failed to update recipes", "error", err, "output", string(output))
		return err
	}

	slog.Info("Extraction recipes updated successfully")
	return nil
}


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
	
	// 1. Collect all potential nodes from all body XPaths
	var allMatchedNodes []*html.Node
	for _, xpath := range recipe.Body {
		nodes, err := htmlquery.QueryAll(root, xpath)
		if err != nil {
			continue
		}
		allMatchedNodes = append(allMatchedNodes, nodes...)
	}

	if len(allMatchedNodes) == 0 {
		return "", fmt.Errorf("recipe applied but no nodes matched")
	}

	// 2. Filter out nodes that are descendants of other matched nodes (Hierarchy De-duplication)
	var finalNodes []*html.Node
	for i, nodeA := range allMatchedNodes {
		isChild := false
		for j, nodeB := range allMatchedNodes {
			if i == j {
				continue
			}
			if isAncestor(nodeB, nodeA) {
				isChild = true
				break
			}
		}
		if !isChild {
			// Check if we already have this exact node (same pointer)
			exists := false
			for _, fn := range finalNodes {
				if fn == nodeA {
					exists = true
					break
				}
			}
			if !exists {
				finalNodes = append(finalNodes, nodeA)
			}
		}
	}

	// 3. Process and clean each top-level node
	var results []string
	seenContent := make(map[string]bool)

	for _, node := range finalNodes {
		// Apply Strip rules INSIDE the node
		for _, stripXpath := range recipe.Strip {
			toStrip, _ := htmlquery.QueryAll(node, stripXpath)
			for _, s := range toStrip {
				if s.Parent != nil {
					s.Parent.RemoveChild(s)
				}
			}
		}

		// Convert to HTML
		htmlContent := htmlquery.OutputHTML(node, true)
		
		// Final de-duplication by normalized text content
		// This handles cases where different XPaths might match different nodes with same text
		plainText := strings.TrimSpace(nodeText(node))
		if plainText == "" && !strings.Contains(htmlContent, "<img") && !strings.Contains(htmlContent, "<iframe") {
			continue // Skip empty blocks that don't have media
		}

		if !seenContent[plainText] || plainText == "" {
			if plainText != "" {
				seenContent[plainText] = true
			}
			results = append(results, htmlContent)
		}
	}

	if len(results) == 0 {
		return "", fmt.Errorf("no content left after cleaning")
	}

	return strings.Join(results, "\n"), nil
}

// nodeText extracts all text from a node and its children
func nodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var buf strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		buf.WriteString(nodeText(c))
	}
	return buf.String()
}

func isAncestor(ancestor, node *html.Node) bool {
	for n := node.Parent; n != nil; n = n.Parent {
		if n == ancestor {
			return true
		}
	}
	return false
}
