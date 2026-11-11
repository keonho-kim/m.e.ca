// Purpose: Run the deterministic browser-bundle packager from the source build.
// Pattern: Development command composition root.
// Usage: webui/scripts/package.ts supplies the built source, output and cache directories.
// Related: devtools/cmd/uiassets/bundle.go, devtools/cmd/uiassets/compression.go.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	input := flag.String("input", "", "built browser directory")
	output := flag.String("output", "", "generated Go bundle file")
	cache := flag.String("cache", "", "owned gzip cache directory")
	flag.Parse()
	if *input == "" || *output == "" || *cache == "" {
		flag.Usage()
		os.Exit(2)
	}
	count, err := packBundle(*input, *output, *cache)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Embedded assets:", count)
}
