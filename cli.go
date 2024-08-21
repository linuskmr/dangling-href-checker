package main

import (
	"flag"
	"fmt"
	"net/url"
	"strings"
)

type CliArgs struct {
	url *url.URL
	showHrefs bool
}

func parseCliArgs() CliArgs {
	showHrefs := flag.Bool("v", false, "Show checked hrefs")
	oldUsage := flag.Usage
	flag.Usage = func() {
		fmt.Println("Debugging tool to find broken links on a website")
		fmt.Println()
		oldUsage()
	}
	flag.Parse()
	urlString := flag.Arg(0)
	if urlString == "" {
		flag.Usage()
		panic("No URL provided")
	}

	if !strings.HasPrefix(urlString, "http") {
		// Just assigning `startUrl.Scheme = "https"` after the URL is parsed does not work because the hostname is interpreted as a path
		urlString = "https://" + urlString
	}
	startUrl, err := url.Parse(urlString)
	if err != nil {
		panic(fmt.Sprintf("Error parsing URL: %s", err))
	}
	return CliArgs{url: startUrl, showHrefs: *showHrefs}
}