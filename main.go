package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"go.khulnsoft.com/go/builder"
)

func main() {
	goos := flag.String("goos", "", "GOOS")
	goarch := flag.String("goarch", "", "GOARCH")
	dst := flag.String("dst", "dist", "build destination")
	readVersion := flag.Bool("read-built-version", false, "If set we'll simply parse go/VERSION.cache and return the Go verison")
	flag.Parse()

	if *readVersion {
		readBuiltVersion()

		return
	}

	if *goos == "" || *goarch == "" || *dst == "" {
		log.Fatalf("missing -dst %q, -goos %q, or -goarch %q", *dst, *goos, *goarch)
	}

	root, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	if err := builder.BuildKhulnsoftGo(*goos, *goarch, root, *dst); err != nil {
		log.Fatalln(err)
	}
}

func readBuiltVersion() {
	str := ""
	if isfile("go/VERSION") {
		// If we're building from a release branch, we use this as the base
		str = readfile("go/VERSION")
		// Then we repeat the replace we do within the src/cmd/dist/build.go
		str = strings.Replace(str, "go1.", "khulnsoft-go1.", 1)
	} else {
		// Otherwise we read the cache file which would be created by the build process
		// if there was no VERSION file present
		str = readfile("go/VERSION.cache")
	}

	// With our patches there must always be an `khulnsoft-go1.xx` version in this string
	// (there may be other bits, like "devel" or "beta" which we don't care about)
	re, err := regexp.Compile("(khulnsoft-go[^ ]+)")
	if err != nil {
		log.Fatalf("Unable to compile regex: %+v", err)
	}
	version := re.FindString(str)
	if version == "" {
		log.Fatalf("Unable to extract version, read: %s", str)
	}

	// In Go 1.21 the time was added as the second line of the VERSION file
	// so we only want the first line
	version, _, _ = strings.Cut(version, "\n")
	version = strings.TrimSpace(version)

	fmt.Println(version)
}

// isfile reports whether p names an existing file.
func isfile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}

// readfile returns the content of the named file.
func readfile(file string) string {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return strings.TrimRight(string(data), " \t\r\n")
}
