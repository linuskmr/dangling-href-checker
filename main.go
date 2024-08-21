package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"
)

var urlRegexp = regexp.MustCompile(`(?:href|src)\s*=\s*["']?([^"'\s<>]+)["']?`)

func main() {
	cliArgs := parseCliArgs()
	if !cliArgs.showHrefs {
		log.SetOutput(io.Discard)
	}
	fmt.Printf("Report for URL %s (%s)\n", cliArgs.url.String(), time.Now().Format(time.RFC3339))
	verifyUrl(cliArgs.url)
}



type Link struct {
	url *url.URL
	href *url.URL
}

func (link Link) String() string {
	return fmt.Sprintf("Link{url: %s, href: %s}", link.url.String(), link.href.String())
}

func verifyUrl(startUrl *url.URL) {
	// Needs to be able to buffer one element (the `startUrl`)
	foundLinks := make(chan Link, 1)
	notFoundErrors := make(chan Link)

	waitGroup := sync.WaitGroup{}
	// Cannot use `map[*url.Url]bool` because this would compare the pointers, not the values
	alreadyCheckedUrls := make(map[string]bool)

	// Start goroutine that checks for already checked urls
	go func() {
		foundLinks <- Link{url: startUrl, href: startUrl}

		/// finish is closed when all threads are done
		finish := make(chan struct{})
		go func() {
			waitGroup.Wait()
			close(finish)
		}()

		out: for {
			select {
			case link := <-foundLinks:
				link.href.Fragment = ""
				if link.url.Hostname() != startUrl.Hostname() {
					// Allows the href to an external page be checked, but not the links on that page
					continue
				}
				if link.href.Scheme != "http" && link.href.Scheme != "https" {
					continue
				}
				if _, ok := alreadyCheckedUrls[link.href.String()]; ok {
					continue
				}
				alreadyCheckedUrls[link.href.String()] = true
				log.Println("Checking", link.href.String())
				waitGroup.Add(1)
				go func() {
					defer waitGroup.Done()
					verifyHref(link, foundLinks, notFoundErrors)
				}()
			case <-finish:
				// There is no thread running anymore, so no one will ever write to the foundLinks channel again
				break out
			}
		}
		close(notFoundErrors)
	}()
	
	numberOfNotFoundErrors := 0
	for notFoundError := range notFoundErrors {
		fmt.Printf("NotFoundError: %s -> %s\n", notFoundError.url.String(), notFoundError.href.String())
		numberOfNotFoundErrors++
	}

	fmt.Printf("Checked %d hrefs, %d errors\n", len(alreadyCheckedUrls), numberOfNotFoundErrors)

	waitGroup.Wait()
}

func verifyHref(hrefToVerify Link, foundLinks chan<- Link, notFoundErrors chan<- Link) {
	response, err := http.Get(hrefToVerify.href.String())
	if err != nil {
		fmt.Printf("Request to %s failed: %s\n", hrefToVerify.href.String(), err)
		notFoundErrors <- hrefToVerify
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Printf("Request to %s failed with status code %d\n", hrefToVerify.href.String(), response.StatusCode)
		notFoundErrors <- hrefToVerify
		return
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Reading response body of %s failed: %s\n", response.Request.URL.String(), err)
		notFoundErrors <- hrefToVerify
	}

	hrefs := urlRegexp.FindAllStringSubmatch(string(body), -1)
	for _, hrefMatch := range hrefs {
		hrefString := hrefMatch[1]
		href, err := url.Parse(hrefString)
		if err != nil {
			fmt.Printf("Error parsing href %s as URL on %s: %s\n", hrefString, response.Request.URL.String(), err)
			continue
		}
		href = hrefToVerify.href.ResolveReference(href)
		foundLinks <- Link{url: hrefToVerify.href, href: href}
	}
}