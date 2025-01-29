package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type CliArgs struct {
	// url is the starting URL that will be checked
	url *url.URL
	// verbose causes all checked hrefs to be printed
	verbose bool
}

func parseCliArgs() CliArgs {
	// Flags/optional arguments
	verbose := flag.Bool("v", false, "Verbose: show checked hrefs")
	oldUsage := flag.Usage
	flag.Usage = func() {
		fmt.Println("Recursively checks a webpage for dangling links (`href`, `src`, `to`), i.e. references to pages that return non `200 OK` status code.")
		fmt.Println("Exits with a status code of 0 if all links were ok or with 1 if there were errors.")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Printf("%s [FLAGS] URL\n", os.Args[0])
		fmt.Println()
		oldUsage()
	}
	flag.Parse()

	// Positional arguments
	urlString := flag.Arg(0)
	if urlString == "" {
		flag.Usage()
		panic("No URL provided")
	}

	if !strings.HasPrefix(urlString, "http") {
		// Adding https has to happen now because just assigning `startUrl.Scheme = "https"`
		// causes the hostname to be interpreted as a path.
		urlString = "https://" + urlString
	}
	startUrl, err := url.Parse(urlString)
	if err != nil {
		panic(fmt.Sprintf("Error parsing URL: %s", err))
	}

	return CliArgs{url: startUrl, verbose: *verbose}
}
