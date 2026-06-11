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

// songHrefPattern matches links to JOYSOUND song pages
// (e.g. /web/search/song/82077) — the song counterpart to
// artistHrefPattern, structurally distinct from it in the same way.
var songHrefPattern = regexp.MustCompile(`^/web/search/song/(\d+)$`)

type joysoundClient struct {
	httpClient *http.Client
}

func newJoysoundClient() *joysoundClient {
	return &joysoundClient{httpClient: &http.Client{Timeout: 10 * time.Second}}
}

// search runs a JOYSOUND keyword search and returns every artist JOYSOUND's
// own search-results page identifies as a match.
func (c *joysoundClient) search(ctx context.Context, keyword string) ([]Artist, error) {
	doc, err := c.fetchSearchDoc(ctx, keyword)
	if err != nil {
		return nil, err
	}
	return findArtists(doc), nil
}

// searchSongs runs a JOYSOUND keyword search and returns every song
// JOYSOUND's own search-results page identifies as a match.
func (c *joysoundClient) searchSongs(ctx context.Context, keyword string) ([]Song, error) {
	doc, err := c.fetchSearchDoc(ctx, keyword)
	if err != nil {
		return nil, err
	}
	return findSongs(doc), nil
}

// fetchSearchDoc runs a JOYSOUND keyword search and returns the parsed HTML
// of the results page, shared by search and searchSongs since both read from
// the same response — JOYSOUND's cross-search returns artist and song
// results in a single page.
func (c *joysoundClient) fetchSearchDoc(ctx context.Context, keyword string) (*html.Node, error) {
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

	return doc, nil
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

// findSongs walks the parsed HTML tree for links matching songHrefPattern
// and returns the song ID (from the link's href), title, and artist (from
// the surrounding markup) for each one found.
func findSongs(n *html.Node) []Song {
	var songs []Song

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			if href, ok := attr(n, "href"); ok {
				if m := songHrefPattern.FindStringSubmatch(href); m != nil {
					if title, artist, ok := songTitleAndArtist(n); ok {
						songs = append(songs, Song{ID: m[1], Title: title, Artist: artist})
					}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)

	return songs
}

// songTitleAndArtist extracts a song result's title and artist from a song
// link's subtree. JOYSOUND renders the title as the first <p> inside the
// link, with the artist name in the next sibling element of that <p>'s
// parent.
func songTitleAndArtist(a *html.Node) (title, artist string, ok bool) {
	p := findFirst(a, "p")
	if p == nil || p.Parent == nil {
		return "", "", false
	}

	title = strings.TrimSpace(textContent(p))

	for sibling := p.Parent.NextSibling; sibling != nil; sibling = sibling.NextSibling {
		if sibling.Type == html.ElementNode {
			artist = strings.TrimSpace(textContent(sibling))
			break
		}
	}

	if title == "" || artist == "" {
		return "", "", false
	}
	return title, artist, true
}

// findFirst returns the first descendant of n (in document order) with the
// given tag name, or nil if there is none.
func findFirst(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if found := findFirst(child, tag); found != nil {
			return found
		}
	}
	return nil
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
