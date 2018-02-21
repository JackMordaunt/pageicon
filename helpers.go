package pageicon

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
)

type slices struct {
	InOrder bool
}

// Left and Right must of equal length and equal contents to be considered equal.
// Order is ignored.
func (s slices) Equal(left, right []string) bool {
	if left == nil && right == nil {
		return true
	}
	if len(left) != len(right) {
		return false
	}

	if !s.InOrder {
		set := map[string]struct{}{}
		for _, l := range left {
			set[l] = struct{}{}
		}
		for _, l := range right {
			if _, ok := set[l]; !ok {
				return false
			}
		}
	} else {
		for ii := 0; ii < len(left); ii++ {
			if left[ii] != right[ii] {
				return false
			}
		}
	}
	return true
}

type slicePrinter struct {
	Slice   interface{}
	Verbose bool
}

func (p slicePrinter) String() string {
	format := "\n\t%v"
	if p.Verbose {
		format = "\n\t%+v"
	}
	v := reflect.ValueOf(p.Slice)
	if v.Kind() != reflect.Slice {
		return ""
	}
	buf := bytes.NewBufferString("[")
	for ii := 0; ii < v.Len(); ii++ {
		buf.WriteString(fmt.Sprintf(format, v.Index(ii)))
		if ii < v.Len()-1 {
			buf.WriteString(",")
		}
	}
	buf.WriteString("\n]")
	return buf.String()
}

func reader(doc string) *bytes.Buffer {
	return bytes.NewBufferString(doc)
}

func fatalf(f string, values ...interface{}) {
	fmt.Printf("Error while testing: "+f, values...)
	os.Exit(1)
}
