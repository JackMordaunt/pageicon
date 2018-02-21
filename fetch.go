package pageicon

import (
	"bytes"
	"io"
	"net/http"
)

// Fetcher fetches a resource given a url.
type Fetcher interface {
	Get(string) (io.Reader, error)
}

// FetcherFunc is a func that satisfies Fetcher interface with a function.
type FetcherFunc func(string) (io.Reader, error)

// Get fetches a resource.
func (f FetcherFunc) Get(url string) (io.Reader, error) {
	return f(url)
}

// SetFetcher allows the package to use a custom resource fetcher.
func SetFetcher(f Fetcher) {
	fetcher = f
}

var fetcher Fetcher = FetcherFunc(func(url string) (io.Reader, error) {
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		resp.Body.Close()
		return nil, err
	}
	resp.Body.Close()
	return buf, nil
})
