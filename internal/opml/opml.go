package opml

import (
	"encoding/xml"
	"io"
)

// OPML models the root element of an OPML file.
type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Body    struct {
		Outlines []Outline `xml:"outline"`
	} `xml:"body"`
}

// Outline models an individual feed or category in an OPML file.
type Outline struct {
	Text     string    `xml:"text,attr"`
	XMLURL   string    `xml:"xmlUrl,attr"`
	HTMLURL  string    `xml:"htmlUrl,attr"`
	Category string    `xml:"title,attr"`
	Outlines []Outline `xml:"outline"`
}

// Parse parses OPML from an io.Reader and returns a flattened list of Outlines with XML URLs.
func Parse(r io.Reader) ([]Outline, error) {
	var o OPML
	if err := xml.NewDecoder(r).Decode(&o); err != nil {
		return nil, err
	}
	return flatten(o.Body.Outlines), nil
}

// flatten recursively extracts outlines with XML URLs from the OPML structure.
func flatten(outlines []Outline) []Outline {
	var result []Outline
	for _, o := range outlines {
		if o.XMLURL != "" {
			result = append(result, o)
		}
		if len(o.Outlines) > 0 {
			result = append(result, flatten(o.Outlines)...)
		}
	}
	return result
}

// Generate serializes a list of outlines into an OPML XML byte slice.
func Generate(outlines []Outline) ([]byte, error) {
	var o OPML
	o.Version = "1.0"
	o.Body.Outlines = outlines
	return xml.MarshalIndent(o, "", "  ")
}
