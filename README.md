# dangling-href-checker

Recursively checks a webpage for dangling links (`href`, `src`, `to`), i.e. references to pages that return non `200 OK` status code.

Exits with a status code of 0 if all links were ok or with 1 if there were errors.



## Usage

Install Go, clone this repository and `cd` into its directory. Then, compile and install the application and run it:

```bash
$ go install .
$ dangling-href-checker -v example.com
Report for URL https://example.com (2025-01-29T19:30:30+01:00)
2025/01/29 19:30:30 Checking https://example.com
2025/01/29 19:30:31 Checking https://www.iana.org/domains/example
Checked 2 hrefs, 0 errors
```