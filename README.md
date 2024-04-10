<h1 align="center">
  TLDS
  <br>
</h1>

`tlds` is a command-line tool to check the status of domain names with various features such as checking domain availability, displaying IP addresses, HTTP status codes, webpage titles, HTTP redirects, favicon hashes, and more.

## Installation

You can install `tlds` using the following command:

```bash
go install github.com/pikpikcu/tlds@latest
```
## USAGE
After installing tlds, you can use it with the following command-line flags:

```bash
tlds -d <base_domain> -F <tld_file> [-o <output_file>] [-ip] [-sc] [-tl] [-fr] [-rl <rate_limit>] [-location] [-favicon] [-ipo <ip_output_file>] [-v]
```
```
Flags:
-d: Base domain name (required)
-F: File containing TLDs (required)
-o, --output: Output file to store domain status
-ip: Display IP address of active domains
-sc, --status-code: Display status code of domains
-tl, --title: Display title of webpages
-fr, --follow-redirects: Follow HTTP redirects
-rl, --rate-limit: Maximum requests to send per second (default 150)
-location: Display response redirect location
-favicon: Display mmh3 hash for '/favicon.ico' file
-ipo: Output file to save IP addresses
-v, --verbose: Verbose mode to display all domains
```
## Example
```
tlds -d example -F wordlists-tlds.txt -o active-domains.txt
```
