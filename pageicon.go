// Package pageicon attemps to find the best icon for a given website based on
// it's markup.
package pageicon

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/pkg/errors"
	filetype "gopkg.in/h2non/filetype.v1"
)

// Infer the icon for the url.
func Infer(url string, preference []string) (*Icon, error) {
	r, err := fetcher.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "fetching url")
	}
	links, err := getIconLinks(url, r)
	if err != nil {
		return nil, errors.Wrap(err, "getting icon links")
	}
	infofln("parsed links: %v", slicePrinter{Slice: links})
	if len(links) < 1 {
		return nil, errors.Wrap(err, "no links found")
	}
	icons, err := downloadIcons(links)
	if err != nil {
		return nil, errors.Wrap(err, "downloading icons")
	}
	if len(icons) == 0 {
		return nil, errors.New("no valid icons")
	}
	infofln("icons downloaded")
	icon := findBestIcon(icons, preference)
	infofln("best icon: %v", icon.Source)
	if icon == nil {
		return nil, errors.New("no best icon")
	}
	return icon, nil
}

// List all the icons links found for a given url.
func List(url string) ([]string, error) {
	r, err := fetcher.Get(url)
	if err != nil {
		return nil, err
	}
	links, err := getIconLinks(url, r)
	if err != nil {
		return nil, err
	}
	return links, nil
}

// Icon represents an icon resource.
type Icon struct {
	Source string
	Data   io.Reader
	Size   int
	Mime   string
	Ext    string
}

// NewFromFile instantiates an Icon from the local file system.
func NewFromFile(path string) (*Icon, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	kind, err := filetype.Match(data)
	if err != nil {
		return nil, err
	}
	icon := &Icon{
		Source: path,
		Ext:    filepath.Ext(path),
		Size:   len(data),
		Data:   bytes.NewBuffer(data),
		Mime:   kind.MIME.Value,
	}
	return icon, nil
}

// findBestIcon finds the biggest icon with the matching extension.
// If no preferences are supplied the largest icon is returned.
func findBestIcon(icons []*Icon, ext []string) *Icon {
	if len(icons) == 0 {
		return nil
	}
	sort.Sort(bySize(icons))
	if len(icons) == 1 {
		return icons[0]
	}
	if len(ext) == 0 {
		return icons[0]
	}
	for _, pref := range ext {
		for _, icon := range icons {
			if icon.Ext == pref {
				return icon
			}
		}
	}
	return icons[0]
}

func getIconLinks(rootURL string, doc io.Reader) ([]string, error) {
	links, err := parseLinks(doc)
	if err != nil {
		return nil, errors.Wrap(err, "parsing document")
	}
	for ii, l := range links {
		if isEmbedded(l) {
			continue
		}
		links[ii] = resolve(rootURL, l)
	}
	return links, nil
}

// resolve path against root, removing any query parameters.
func resolve(root, path string) string {
	if root == "" {
		return ""
	}
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http") {
		return path
	}
	if !strings.HasPrefix(root, "http") {
		root = "https://" + root
	}
	rootURL, _ := url.Parse(root)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	pathURL, _ := url.Parse(path)
	u := fmt.Sprintf("https://%s%s", rootURL.Hostname(), pathURL.Path)
	clean, _ := url.Parse(u)
	return rootURL.ResolveReference(clean).String()
}

// download icons in parallel.
func downloadIcons(links []string) ([]*Icon, error) {
	downloaded := make(chan *Icon)
	failed := make(chan string)
	done := make(chan []*Icon)
	collectIcons := func(downloaded chan *Icon, done chan []*Icon) {
		icons := []*Icon{}
		for icon := range downloaded {
			icons = append(icons, icon)
		}
		done <- icons
	}
	logMessages := func(messages chan string) {
		for msg := range messages {
			infofln(msg)
		}
	}
	download := func(downloaded chan *Icon, failed chan string) {
		downloading := &sync.WaitGroup{}
		for ii, l := range links {
			ii := ii
			l := l
			downloading.Add(1)
			go func() {
				defer downloading.Done()
				icon, err := downloadIcon(l)
				if err != nil {
					failed <- fmt.Sprintf(
						"download failed for %d: %s: %s",
						ii,
						l,
						truncate(fmt.Sprintf("%s", err)),
					)
					return
				}
				downloaded <- icon
			}()
		}
		downloading.Wait()
	}
	go collectIcons(downloaded, done)
	go logMessages(failed)
	download(downloaded, failed)
	close(downloaded)
	close(failed)
	icons := <-done
	close(done)
	return icons, nil
}

// downloadIcon will fetch icon image from the link or decode the icon image if
// it's embedded in the link.
func downloadIcon(link string) (*Icon, error) {
	buf := bytes.NewBuffer(nil)
	if isEmbedded(link) {
		base64Data := link[strings.Index(link, "base64,")+len("base64,"):]
		decoder := base64.NewDecoder(
			base64.StdEncoding,
			bytes.NewBufferString(base64Data),
		)
		data, err := ioutil.ReadAll(decoder)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewBuffer(data)
	} else {
		resp, err := fetcher.Get(link)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(buf, resp); err != nil {
			return nil, err
		}
	}
	kind, err := filetype.Match(buf.Bytes())
	if err != nil {
		return nil, err
	}
	icon := &Icon{
		Source: link,
		Size:   buf.Len(),
		Data:   buf,
		Mime:   kind.MIME.Value,
		Ext:    kind.Extension,
	}
	return icon, nil
}

func isEmbedded(link string) bool {
	return strings.HasPrefix(link, "data:image/")
}

// truncate is an error message helper.
// Printing urls becomes unweildy when they include an embedded image, so you
// truncate the url to make the error message more clear.
func truncate(str string) string {
	if len(str) < 100 {
		return str
	}
	return string(append([]byte(str)[:100], []byte("...")...))
}
