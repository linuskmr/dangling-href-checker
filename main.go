package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"
)

var urlRegexp = regexp.MustCompile(`(?:to|src|href)\s*=\s*["']?([^"'\s<>]+)["']?`)

func main() {
	cliArgs := parseCliArgs()
	if !cliArgs.verbose {
		log.SetOutput(io.Discard)
	}
	fmt.Printf("Report for URL %s (%s)\n", cliArgs.url.String(), time.Now().Format(time.RFC3339))
	numberOfNotFoundErrors := checkWebpage(cliArgs.url)
	exitCode := 0
	if numberOfNotFoundErrors != 0 {
		exitCode = 1
	}
	os.Exit(exitCode)
}

// Link is a link on the page `from` that links to `to`.
type Link struct {
	from *url.URL
	to   *url.URL
}

func (link Link) String() string {
	return fmt.Sprintf("Link{from: %s, to: %s}", link.from.String(), link.to.String())
}

// checkWebpage searches for hrefs on the page `startUrl` and calls `verifyLink` for each of them,
// thereby verifying that they are not 404-ing.
func checkWebpage(startUrl *url.URL) int {
	// foundLinks needs to be able to buffer one element: the `startUrl` added later
	foundLinks := make(chan Link, 1)
	notFoundErrors := make(chan Link)

	// This wait group is used to wait for all link checking threads to finish
	waitGroup := sync.WaitGroup{}

	// alreadyCheckedUrls prevents looking up a URL multiple times
	// Cannot use `map[*from.Url]bool` because this would compare the pointers, not the values
	alreadyCheckedUrls := make(map[string]bool)

	// Start goroutine that checks for already checked urls
	go func() {
		foundLinks <- Link{from: startUrl, to: startUrl}

		// If the wait group indicates that all link-checking threads are finished,
		// close the foundLinks channel to end the execution of the loop below and thereby also this thread.
		go func() {
			waitGroup.Wait()
			close(foundLinks)
		}()

		// Reads links from `foundLinks` and call `verifyLink` on them
		for link := range foundLinks {
			link.to.Fragment = ""
			isExternalWebpage := link.from.Hostname() != startUrl.Hostname()
			if isExternalWebpage {
				// Only check the domain supposed to be checked and not the whole internet
				continue
			}
			if link.to.Scheme != "http" && link.to.Scheme != "https" {
				continue
			}
			if _, ok := alreadyCheckedUrls[link.to.String()]; ok {
				continue
			}
			alreadyCheckedUrls[link.to.String()] = true
			log.Println("Checking", link.to.String())
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				verifyLink(link, foundLinks, notFoundErrors)
			}()
		}
		close(notFoundErrors)
	}()

	return printNotFoundErrors(notFoundErrors, alreadyCheckedUrls)
}

// verifyLink checks Link.to. In case it is 404-ing, sends it to notFoundErrors.
// All links on the page of Link.to are sent to foundLinks to be checked as well.
func verifyLink(link Link, foundLinks chan<- Link, notFoundErrors chan<- Link) {
	response, err := http.Get(link.to.String())
	if err != nil {
		fmt.Printf("Request to %s failed: %s\n", link.to.String(), err)
		notFoundErrors <- link
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Cannot close http response body: %v\n", err)
		}
	}(response.Body)

	if response.StatusCode != http.StatusOK {
		log.Printf("Request to %s failed with status code %d\n", link.to.String(), response.StatusCode)
		notFoundErrors <- link
		return
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Reading response body of %s failed: %s\n", response.Request.URL.String(), err)
		notFoundErrors <- link
	}

	hrefs := urlRegexp.FindAllStringSubmatch(string(body), -1)
	for _, hrefMatch := range hrefs {
		hrefString := hrefMatch[1]
		href, err := url.Parse(hrefString)
		if err != nil {
			fmt.Printf("Error parsing to %s as URL on %s: %s\n", hrefString, response.Request.URL.String(), err)
			continue
		}
		href = link.to.ResolveReference(href)
		foundLinks <- Link{from: link.to, to: href}
	}
}

func printNotFoundErrors(notFoundErrors chan Link, alreadyCheckedUrls map[string]bool) int {
	numberOfNotFoundErrors := 0
	for notFoundError := range notFoundErrors {
		fmt.Printf("NotFoundError: %s -> %s\n", notFoundError.from.String(), notFoundError.to.String())
		numberOfNotFoundErrors++
	}

	fmt.Printf("Checked %d hrefs, %d errors\n", len(alreadyCheckedUrls), numberOfNotFoundErrors)

	return numberOfNotFoundErrors
}
