package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jackmordaunt/pageicon"
)

func main() {
	if len(os.Args) < 2 {
		fatalf("Specify a url to list the websites's icons.")
	}
	url := os.Args[1]
	if !strings.HasPrefix(url, "http") {
		url = fmt.Sprintf("https://%s", url)
	}
	icons, err := pageicon.List(url)
	if err != nil {
		fatalf("Error: %v", err)
	}
	if len(icons) == 0 {
		fmt.Printf("no icons found\n")
		os.Exit(0)
	}
	for ii, icon := range icons {
		fmt.Printf("%d: %s\n", ii, icon)
	}
	fmt.Printf("\n")
}

func fatalf(f string, v ...interface{}) {
	fmt.Fprintf(os.Stdout, f, v...)
	os.Exit(1)
}
