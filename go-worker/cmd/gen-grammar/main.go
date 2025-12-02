package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"unicode"
)

type Child struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Category struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Children []*Child `json:"children,omitempty"`
	Level    int      `json:"level"`
}

type BaseCategory struct {
	Name       string      `json:"name"`
	Categories []*Category `json:"categories"`
}

type Root struct {
	Verticals []*BaseCategory `json:"verticals"`
}

func main() {
	fmt.Println("Fetching latest taxonomy from Shopify...")
	resp, err := http.Get("https://raw.githubusercontent.com/Shopify/product-taxonomy/refs/heads/main/dist/en/taxonomy.json")

	if err != nil {
		fmt.Printf("Error fetching taxonomy: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return
	}

	var root Root
	err = json.Unmarshal(body, &root)
	if err != nil {
		fmt.Printf("Error unmarshalling taxonomy: %v\n", err)
		return
	}

	// Only using apparel vertical
	apparel, err := findBaseCategory(&root, "Apparel & Accessories")
	if err != nil {
		fmt.Printf("Error finding base category: %v\n", err)
		return
	}

	fmt.Printf("Found %d categories in apparel vertical\n", len(apparel.Categories))

	gbnf := `root ::= "{" ws "\"title\":" ws string ws "," ws "\"description\":" ws string ws "," ws "\"taxonomy\":" ws taxonomy ws "}"
ws ::= [ \t\n\r]*
string ::= "\"" char* "\""
char ::= [^"\\] | "\\" ["\\/bfnrt]
seperator ::= " > "
taxonomy ::= "\"" taxonomy-inner "\""
`

	for _, category := range apparel.Categories {
		gbnf += generateRule(category)
	}
	os.WriteFile("../docs/taxonomy.gbnf", []byte(gbnf), 0644)
	fmt.Println("Taxonomy grammar file written to ../docs/taxonomy.gbnf")
}

func generateRule(category *Category) string {
	gbnfOutput := "taxonomy"

	if category.Level > 0 {
		gbnfOutput += "-" + clean(category.Name)
	}else {
		gbnfOutput += "-inner"
	}

	gbnfOutput += " ::= \"" + category.Name + "\" "

	if len(category.Children) > 0 {
		gbnfOutput += "(seperator taxonomy-" + clean(category.Name) + "-children)"
		if (category.Level > 0) {
			gbnfOutput += "? \n\n"
		}else {
			gbnfOutput += "\n\n"
		}
		gbnfOutput += "taxonomy-" + clean(category.Name) + "-children ::= (\n"
		for _, child := range category.Children {
			gbnfOutput += "\ttaxonomy-" + clean(child.Name)
			if child != category.Children[len(category.Children)-1] {
				gbnfOutput += " | \n"
			}
		}
		gbnfOutput += "\n)\n\n"
	} else {
		gbnfOutput += "\n\n"
	}

	return gbnfOutput
}

func findBaseCategory(root *Root, categoryID string) (*BaseCategory, error) {
	for _, vertical := range root.Verticals {
		if vertical.Name == categoryID {
			return vertical, nil
		}
	}
	return nil, fmt.Errorf("base category not found: %s", categoryID)
}

func clean(s string) string {
	var b strings.Builder
	for _, r := range s {
		if  (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}else if r >= 'A' && r <= 'Z' {
			b.WriteRune(unicode.ToLower(r))
		}else if (b.String()[len(b.String())-1] != '-') {
			b.WriteRune('-')
		}
	}
	return b.String()
}
