package pageicon

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/pkg/errors"
)

var equal = reflect.DeepEqual

func TestGetIconLinks(t *testing.T) {
	t.Parallel()
	type params struct {
		rootURL string
		dom     io.Reader
	}
	type returns struct {
		links []string
		err   bool
	}
	type fixture struct {
		desc    string
		params  params
		returns returns
	}
	tests := []fixture{
		{
			desc: "Link tags.",
			params: params{
				rootURL: "https://valid.com",
				dom: reader(`
<html>
	<head>
		<link href="/icon.png">
		<link href="/icon.jpg">
		<link rel="icon" href="/any.ico">
		<link rel="apple-touch" href="data:image/png;base64,embeddedimage">
		<meta property="og:image" content="/this/is/an/icon.jpg">
		<meta property="og:image" content="/this/is/an/icon-2.jpg">
		<meta property="og:image" content="/images/fb_icon_325x325.png">
		<link href="/icon.mp4">
	</head>
</html>`),
			},
			returns: returns{
				links: []string{
					"https://valid.com/icon.png",
					"https://valid.com/icon.jpg",
					"https://valid.com/any.ico",
					"https://valid.com/this/is/an/icon.jpg",
					"https://valid.com/this/is/an/icon-2.jpg",
					"data:image/png;base64,embeddedimage",
					"https://valid.com/images/fb_icon_325x325.png",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(st *testing.T) {
			want := tt.returns.links
			wantErr := tt.returns.err
			got, err := getIconLinks(tt.params.rootURL, tt.params.dom)
			if wantErr && err == nil {
				st.Fatalf("wanted error, got nil")
			}
			if !wantErr && err != nil {
				st.Fatalf("unexpected error: %v", err)
			}
			s := slices{}
			if !s.Equal(got, want) {
				st.Fatalf("%s \nwant=%s \ngot=%s \n",
					tt.desc,
					slicePrinter{
						Slice:   want,
						Verbose: true,
					},
					slicePrinter{
						Slice:   got,
						Verbose: true,
					},
				)
			}
		})
	}
}

func TestFindBestIcon(t *testing.T) {
	t.Parallel()
	var biggest = 999
	var middle = 500
	var smallest = 1
	type fixture struct {
		desc  string
		icons []*Icon
		ext   []string
		want  *Icon
	}
	tests := []fixture{
		{
			desc:  "Nil icon list.",
			icons: nil,
			ext:   nil,
			want:  nil,
		},
		{
			desc:  "Empty icon list.",
			icons: []*Icon{},
			ext:   nil,
			want:  nil,
		},
		{
			desc: "In order, no preferences.",
			icons: []*Icon{
				{Size: biggest},
				{Size: middle},
				{Size: smallest},
			},
			ext: nil,
			want: &Icon{
				Size: biggest,
			},
		},
		{
			desc: "Out of order, biggest first, no preferences.",
			icons: []*Icon{
				{Size: biggest},
				{Size: smallest},
				{Size: middle},
			},
			ext: nil,
			want: &Icon{
				Size: biggest,
			},
		},
		{
			desc: "Out of order, biggest last, no preferences.",
			icons: []*Icon{
				{Size: middle},
				{Size: smallest},
				{Size: biggest},
			},
			ext: nil,
			want: &Icon{
				Size: biggest,
			},
		},
		{
			desc: "png preference, with 1 'png' icon.",
			icons: []*Icon{
				{Size: middle, Ext: "jpg"},
				{Size: smallest, Ext: "png"},
				{Size: biggest, Ext: "jpg"},
			},
			ext: []string{"png"},
			want: &Icon{
				Size: smallest,
				Ext:  "png",
			},
		},
		{
			desc: "png preference, with > 1 'png' icons.",
			icons: []*Icon{
				{Size: middle, Ext: "png"},
				{Size: smallest, Ext: "png"},
				{Size: biggest, Ext: "jpg"},
			},
			ext: []string{"png"},
			want: &Icon{
				Size: middle,
				Ext:  "png",
			},
		},
		{
			desc: "Multiple preferences.",
			icons: []*Icon{
				{Size: middle, Ext: "jpg"},
				{Size: smallest, Ext: "jpg"},
				{Size: biggest, Ext: "jpg"},
			},
			ext: []string{"png", "jpg"},
			want: &Icon{
				Size: biggest,
				Ext:  "jpg",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(st *testing.T) {
			got := findBestIcon(tt.icons, tt.ext)
			if !equal(got, tt.want) {
				st.Errorf("want=%v, got=%v\n", tt.want, got)
			}
		})
	}
}

// TestDownloadIcon does not test for well formed urls.
// If a url is poorly formed the fetcher must deal with it, most likely as a
// failure.
// The focus of the testing here is:
// - embedded icons (inside link string)
// - fetching the icon if not embedded
// - correct Mime detection
// 	- png
// 	- ico
func TestDownloadIcon(t *testing.T) {
	t.Parallel()
	defaultFetcher := fetcher
	fetcher = FetcherFunc(func(url string) (io.Reader, error) {
		return _png(_rect(16, 16)), nil
	})
	defer func() {
		fetcher = defaultFetcher
	}()
	tests := []struct {
		desc string
		link string
		want *Icon
	}{
		{
			"linked resource",
			"https://example.com",
			&Icon{
				Source: "https://example.com",
				Size:   _size(_png(_rect(16, 16))),
				Data:   _png(_rect(16, 16)),
				Mime:   "image/png",
				Ext:    "png",
			},
		},
		{
			"embedded resource",
			_embedPNG(_png(_rect(16, 16))),
			&Icon{
				Source: _embedPNG(_png(_rect(16, 16))),
				Size:   _size(_png(_rect(16, 16))),
				Data:   _png(_rect(16, 16)),
				Mime:   "image/png",
				Ext:    "png",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(st *testing.T) {
			icon, err := downloadIcon(tt.link)
			if err != nil {
				st.Fatalf("unexpected error: %v", err)
			}
			if !iconEqual(icon, tt.want) {
				st.Fatalf("\nwant=%+v, \ngot=%+v", tt.want, icon)
			}
		})
	}
}

func _rect(w, h int) image.Image {
	return image.Rect(0, 0, w, h)
}

func _png(img image.Image) io.Reader {
	buf := bytes.NewBuffer(nil)
	if err := png.Encode(buf, img); err != nil {
		panic(errors.Wrapf(err, "encoding png"))
	}
	return buf
}

func _embedPNG(r io.Reader) string {
	prefix := "data:image/png;base64,"
	buf := bytes.NewBufferString(prefix)
	enc := base64.NewEncoder(base64.StdEncoding, buf)
	defer enc.Close()
	if _, err := io.Copy(enc, r); err != nil {
		panic(err)
	}
	s := buf.String()
	return s
}

func _size(r io.Reader) int {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return len(data)
}

func iconEqual(left, right *Icon) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil && right != nil {
		return false
	}
	if left != nil && right == nil {
		return false
	}
	if left.Ext != right.Ext {
		return false
	}
	if left.Size != right.Size {
		return false
	}
	if left.Source != right.Source {
		return false
	}
	if left.Mime != right.Mime {
		return false
	}
	lb, err := ioutil.ReadAll(left.Data)
	if err != nil {
		panic(err)
	}
	rb, err := ioutil.ReadAll(right.Data)
	if err != nil {
		panic(err)
	}
	return reflect.DeepEqual(lb, rb)
}

func TestResolve(t *testing.T) {
	t.Parallel()
	tests := []struct {
		desc string

		root string
		path string

		want string
	}{
		{
			"standard url",
			"https://www.example.com",
			"/path/to/icon.png",
			"https://www.example.com/path/to/icon.png",
		},
		{
			"trailing slash",
			"https://www.example.com/",
			"/path/to/icon.png",
			"https://www.example.com/path/to/icon.png",
		},
		{
			"no preceding slash",
			"https://www.example.com",
			"path/to/icon.png",
			"https://www.example.com/path/to/icon.png",
		},
		{
			"no scheme",
			"www.example.com",
			"/path/to/icon.png",
			"https://www.example.com/path/to/icon.png",
		},
		{
			"root more than hostname",
			"https://www.example.com/extra",
			"/path/to/icon.png",
			"https://www.example.com/path/to/icon.png",
		},
		{
			"empty path",
			"https://www.example.com",
			"",
			"",
		},
		{
			"empty root",
			"",
			"/path/to/icon.png",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(st *testing.T) {
			got := resolve(tt.root, tt.path)
			if got != tt.want {
				st.Fatalf("\nwant=%s \ngot=%s", tt.want, got)
			}
		})
	}
}
