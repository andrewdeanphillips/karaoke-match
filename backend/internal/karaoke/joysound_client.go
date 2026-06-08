package karaoke

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const searchURL = "https://www.joysound.com/web/search/cross"

// artistHrefPattern matches links to JOYSOUND artist pages
// (e.g. /web/search/artist/62831). JOYSOUND's search-results page renders
// these links exclusively for results it classifies as artists, keeping them
// structurally distinct from song results in the same response — so finding
// this pattern is sufficient to identify an artist match, with no further
// classification needed on our part.
var artistHrefPattern = regexp.MustCompile(`^/web/search/artist/(\d+)$`)

type joysoundClient struct {
	httpClient *http.Client
}

func newJoysoundClient() *joysoundClient {
	return &joysoundClient{httpClient: &http.Client{Timeout: 10 * time.Second}}
}

// search runs a JOYSOUND keyword search and returns every artist JOYSOUND's
// own search-results page identifies as a match.
func (c *joysoundClient) search(ctx context.Context, keyword string) ([]Artist, error) {
	reqURL := searchURL + "?match=1&keyword=" + url.QueryEscape(keyword)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building JOYSOUND search request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting JOYSOUND search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JOYSOUND search request failed with status %d", resp.StatusCode)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing JOYSOUND search response: %w", err)
	}

	return findArtists(doc), nil
}

// findArtists walks the parsed HTML tree for links matching artistHrefPattern
// and returns the artist ID (from the link's href) and display name (from
// the link's visible text) for each one found.
func findArtists(n *html.Node) []Artist {
	var artists []Artist

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			if href, ok := attr(n, "href"); ok {
				if m := artistHrefPattern.FindStringSubmatch(href); m != nil {
					if name := strings.TrimSpace(textContent(n)); name != "" {
						artists = append(artists, Artist{ID: m[1], Name: name})
					}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)

	return artists
}

func attr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		sb.WriteString(textContent(child))
	}
	return sb.String()
}
