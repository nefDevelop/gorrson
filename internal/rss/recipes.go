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
	"sync"
	"time"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

var (
	updateMu       sync.Mutex
	lastUpdate     time.Time
	updateInterval = 30 * time.Minute // Don't update more than once every 30 mins
)

// UpdateRecipes performs a git pull on the ftr-site-config repository to get the latest rules.
func UpdateRecipes() error {
	updateMu.Lock()
	defer updateMu.Unlock()

	if time.Since(lastUpdate) < updateInterval {
		return nil
	}

	basePath := "data/site-config/ftr-site-config"
	// Check if the directory exists and is a git repo
	if _, err := os.Stat(filepath.Join(basePath, ".git")); os.IsNotExist(err) {
		slog.Info("Recipes repository not found, cloning...", "path", basePath)
		cmd := exec.Command("git", "clone", "https://github.com/fivefilters/ftr-site-config.git", basePath)
		if output, err := cmd.CombinedOutput(); err != nil {
			slog.Error("Failed to clone recipes", "error", err, "output", string(output))
			return err
		}
		slog.Info("Extraction recipes cloned successfully")
		lastUpdate = time.Now()
		return nil
	}

	slog.Info("Updating extraction recipes from repository...")
	cmd := exec.Command("git", "-C", basePath, "pull", "origin", "master")
	if output, err := cmd.CombinedOutput(); err != nil {
		slog.Error("Failed to update recipes", "error", err, "output", string(output))
		return err
	}

	slog.Info("Extraction recipes updated successfully")
	lastUpdate = time.Now()
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

// ApplyRecipe applies the rules to an HTML document and returns a container node with matches.
func ApplyRecipe(root *html.Node, recipe *Recipe) (*html.Node, error) {
	if recipe == nil {
		return nil, fmt.Errorf("nil recipe")
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
		return nil, fmt.Errorf("recipe applied but no nodes matched")
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

	// 3. Create a container node to hold all results
	container := &html.Node{
		Type: html.ElementNode,
		Data: "div",
	}

	for _, node := range finalNodes {
		// Apply Strip rules INSIDE the node
		for _, stripXPath := range recipe.Strip {
			nodesToRemove, err := htmlquery.QueryAll(node, stripXPath)
			if err == nil {
				for _, n := range nodesToRemove {
					if n.Parent != nil {
						n.Parent.RemoveChild(n)
					}
				}
			}
		}

		// Clone and append to container
		cloned := cloneNode(node)
		container.AppendChild(cloned)
	}

	return container, nil
}

func cloneNode(n *html.Node) *html.Node {
	m := &html.Node{
		Type:     n.Type,
		DataAtom: n.DataAtom,
		Data:     n.Data,
		Attr:     make([]html.Attribute, len(n.Attr)),
	}
	copy(m.Attr, n.Attr)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		m.AppendChild(cloneNode(c))
	}
	return m
}

func isAncestor(ancestor, child *html.Node) bool {
	for n := child.Parent; n != nil; n = n.Parent {
		if n == ancestor {
			return true
		}
	}
	return false
}
