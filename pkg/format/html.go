package format

import (
	"encoding/base64"
	"strings"

	"golang.org/x/net/html"
)

func ExtractHTMLTitleFromB64Html(body []byte) string {
	decoded, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		decoded = body
	}

	content := string(decoded)
	contentLower := strings.ToLower(content)

	if !strings.Contains(contentLower, "<html") {
		return ""
	}

	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return ""
	}

	var title string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			title = n.FirstChild.Data
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return title
}
