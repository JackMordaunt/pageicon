package pageicon

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

// token wraps html.Token with helper methods.
type token struct {
	html.Token
}

// Attr lookup attributes associated with this token.
func (t *token) Attr(name string) (string, bool) {
	for _, a := range t.Token.Attr {
		if a.Key == name {
			return a.Val, true
		}
	}
	return "", false
}

func parseLinks(r io.Reader) ([]string, error) {
	var links []string
	z := html.NewTokenizer(r)
loop:
	for {
		tt := z.Next()
		if err := z.Err(); err != nil && err != io.EOF {
			return nil, err
		}
		switch tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			t := token{z.Token()}
			switch t.Data {
			case "link":
				if link, ok := iconFromLink(t); ok {
					links = append(links, link)
				}
			case "meta":
				if link, ok := iconFromMeta(t); ok {
					links = append(links, link)
				}
			}
		case html.ErrorToken:
			break loop
		}
	}
	return links, nil
}

func iconFromLink(t token) (string, bool) {
	if href, ok := t.Attr("href"); ok {
		if strings.Contains(href, "icon.") {
			isPng := strings.HasSuffix(href, ".png")
			isJpg := strings.HasSuffix(href, ".jpg")
			if isPng || isJpg {
				return href, true
			}
		}
	}
	if rel, ok := t.Attr("rel"); ok {
		if strings.Contains(rel, "apple-touch") || strings.Contains(rel, "icon") {
			if href, ok := t.Attr("href"); ok {
				return href, true
			}
		}
	}
	return "", false
}

func iconFromMeta(t token) (string, bool) {
	if property, ok := t.Attr("property"); !ok || property != "og:image" {
		return "", false
	}
	if graphImageURL, ok := t.Attr("content"); ok {
		return graphImageURL, true
	}
	return "", false
}
