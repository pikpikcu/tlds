package main

import (
    "bufio"
    "flag"
    "fmt"
    "hash/fnv"
    "net"
    "net/http"
    "net/url"
    "os"
    "strings"
    "sync"
    "time"

    "github.com/PuerkitoBio/goquery"
    "github.com/fatih/color"
    "github.com/spaolacci/murmur3"
)

func main() {
    var baseDomain string
    var tldFile string
    var outputFile string
    var outputIPFile string
    var showIP bool
    var verbose bool
    var showStatusCode bool
    var showTitle bool
    var numThreads int
    var rateLimit int
    var followRedirects bool
    var maxRedirects int
    var followHostRedirects bool
    var showLocation bool
    var showFavicon bool

    // Define flags
    flag.StringVar(&baseDomain, "d", "", "Base domain name")
    flag.StringVar(&tldFile, "F", "", "File containing TLDs")
    flag.StringVar(&outputFile, "o", "", "Output file to store domain status")
    flag.StringVar(&outputIPFile, "ipo", "", "Output file to save IP addresses")
    flag.BoolVar(&showIP, "ip", false, "Display IP address of active domains")
    flag.BoolVar(&verbose, "v", false, "Verbose mode to display all domains")
    flag.BoolVar(&showStatusCode, "sc", false, "Display status code of domains")
    flag.BoolVar(&showStatusCode, "status-code", false, "Display status code of domains")
    flag.BoolVar(&showTitle, "tl", false, "Display title of webpages")
    flag.BoolVar(&showTitle, "title", false, "Display title of webpages")
    flag.IntVar(&numThreads, "t", 50, "Number of threads to use")
    flag.IntVar(&numThreads, "threads", 50, "Number of threads to use")
    flag.IntVar(&rateLimit, "rl", 150, "Maximum requests to send per second")
    flag.IntVar(&rateLimit, "rate-limit", 150, "Maximum requests to send per second")
    flag.BoolVar(&followRedirects, "fr", false, "Follow HTTP redirects")
    flag.IntVar(&maxRedirects, "maxr", 10, "Maximum number of redirects to follow per host")
    flag.BoolVar(&followHostRedirects, "fhr", false, "Follow redirects on the same host")
    flag.BoolVar(&showLocation, "location", false, "Display response redirect location")
    flag.BoolVar(&showFavicon, "favicon", false, "Display mmh3 hash for '/favicon.ico' file")

    // Set usage function
    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage:\n")
        fmt.Fprintf(os.Stderr, "  tlds [flags]\n\n")
        fmt.Fprintf(os.Stderr, "Flags:\n")

        // INPUT
        fmt.Fprintf(os.Stderr, "INPUT\n")
        fmt.Fprintf(os.Stderr, "  -d                      string Base domain name\n")
        fmt.Fprintf(os.Stderr, "  -F                      string File containing TLDs\n")
        fmt.Fprintf(os.Stderr, "\nPROBES:\n")
        fmt.Fprintf(os.Stderr, "  -ip                     Display IP address of active domains\n")
        fmt.Fprintf(os.Stderr, "  -sc, -status-code       Display status code of domains\n")
        fmt.Fprintf(os.Stderr, "  -title, -tl             Display title of webpages\n")
        fmt.Fprintf(os.Stderr, "  -favicon                Display mmh3 hash for '/favicon.ico' file\n")
        fmt.Fprintf(os.Stderr, "  -location               Display response redirect location\n")
        fmt.Fprintf(os.Stderr, "  -fr                     Follow HTTP redirects\n")
        fmt.Fprintf(os.Stderr, "  -fhr                    Follow redirects on the same host\n")
        fmt.Fprintf(os.Stderr, "  -maxr int               Maximum number of redirects to follow per host (default 10)\n\n")

        fmt.Fprintf(os.Stderr, "OUTPUT:\n")
        fmt.Fprintf(os.Stderr, "  -o,-output string Output file to store domain status\n")
        fmt.Fprintf(os.Stderr, "  -ipo string       Output file to save IP addresses\n\n")

        fmt.Fprintf(os.Stderr, "RATE-LIMIT:\n")
        fmt.Fprintf(os.Stderr, "  -rl, -rate-limit int    Maximum requests to send per second (default 150)\n")
        fmt.Fprintf(os.Stderr, "  -t,  -threads int       Number of threads to use (default 50)\n\n")

        fmt.Fprintf(os.Stderr, "DEBUG:\n")
        fmt.Fprintf(os.Stderr, "  -v,  -verbose           Verbose mode to display all domains\n")
    }

    // Parse flags
    flag.Parse()

    // Validate required flags
    if baseDomain == "" || tldFile == "" {
        flag.Usage()
        return
    }

    // Reading TLD file
    tldList, err := readLines(tldFile)
    if err != nil {
        fmt.Println("Error reading TLD file:", err)
        return
    }

    // Create output file if specified
    var output *os.File
    if outputFile != "" {
        output, err = os.Create(outputFile)
        if err != nil {
            fmt.Println("Error creating output file:", err)
            return
        }
        defer output.Close()
    }

    // Create IP output file if specified
    var outputIP *os.File
    if outputIPFile != "" {
        outputIP, err = os.Create(outputIPFile)
        if err != nil {
            fmt.Println("Error creating IP output file:", err)
            return
        }
        defer outputIP.Close()
    }

    // Channel to communicate between worker goroutines and main goroutine
    domainChannel := make(chan string)

    // Semaphore to control rate limit
    semaphore := make(chan struct{}, rateLimit)

    // WaitGroup to wait for all goroutines to finish
    var wg sync.WaitGroup

    // Launching worker goroutines
    for i := 0; i < numThreads; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for domain := range domainChannel {
                semaphore <- struct{}{} // Acquire token from semaphore
                active, ip, statusCode, title, location, faviconHash := checkDomainStatus(domain, followRedirects, maxRedirects, followHostRedirects, showTitle, showLocation, showFavicon)
                if active {
                    fmt.Print(domain, " ")
                    color.New(color.FgGreen).Printf("[Active] ")
                    if showIP {
                        color.New(color.FgYellow).Printf("[IP:%s] ", ip)
                    }
                    if showStatusCode {
                        color.New(color.FgMagenta).Printf("[Status Code:%s] ", statusCode)
                    }
                    if showTitle {
                        color.New(color.FgGreen).Printf("[Title:%s] ", title)
                    }
                    if showLocation {
                        color.New(color.FgBlue).Printf("[Location:%s] ", location)
                    }
                    if showFavicon {
                        fmt.Printf("[Favicon Hash:%s] ", faviconHash)
                    }
                    fmt.Println()
                    // Writing to output file if specified
                    if outputFile != "" {
                        writeOutput(output, domain, ip, statusCode, title, location, faviconHash, showIP, showStatusCode, showTitle, showLocation, showFavicon)
                    }
                    // Writing to IP output file if specified
                    if outputIPFile != "" {
                        writeIP(outputIP, ip)
                    }
                } else if verbose {
                    fmt.Print(domain, " ")
                    color.New(color.FgRed).Printf("[Not Active]\n")
                    // Writing to output file if specified
                    if outputFile != "" {
                        writeOutput(output, domain, "", statusCode, "", "", "", showIP, showStatusCode, showTitle, showLocation, showFavicon)
                    }
                }
                <-semaphore // Release token to semaphore
            }
        }()
    }

    // Sending domains to the channel
    for _, tld := range tldList {
        domain := baseDomain + "." + tld
        domainChannel <- domain
    }

    // Closing the channel to signal that all domains have been sent
    close(domainChannel)

    // Waiting for all goroutines to finish
    wg.Wait()

    if outputFile != "" {
        fmt.Println("Domains status written to", outputFile)
    }
    if outputIPFile != "" {
        fmt.Println("IP addresses saved to", outputIPFile)
    }
}

// Function to write output to file
func writeOutput(file *os.File, domain, ip, statusCode, title, location, faviconHash string, showIP, showStatusCode, showTitle, showLocation, showFavicon bool) {
    _, err := file.WriteString(domain + " [Active]")
    if err != nil {
        fmt.Println("Error writing to output file:", err)
        return
    }
    if showIP {
        _, err = file.WriteString(" [IP: " + ip + "]")
    }
    if showStatusCode {
        _, err = file.WriteString(" [Status Code: " + statusCode + "]")
    }
    if showTitle {
        _, err = file.WriteString(fmt.Sprintf(" [Title:%s]", title))
    }
    if showLocation {
        _, err = file.WriteString(fmt.Sprintf(" [Location:%s]", location))
    }
    if showFavicon {
        _, err = file.WriteString(fmt.Sprintf(" [Favicon Hash:%s]", faviconHash))
    }
    _, err = file.WriteString("\n")
    if err != nil {
        fmt.Println("Error writing to output file:", err)
        return
    }
}

// Function to write IP address to file
func writeIP(file *os.File, ip string) {
    _, err := file.WriteString(ip + "\n")
    if err != nil {
        fmt.Println("Error writing to IP output file:", err)
        return
    }
}

// Function to check if domain is active and get its IP address, status code, title, location, and favicon hash
func checkDomainStatus(domain string, followRedirects bool, maxRedirects int, followHostRedirects bool, showTitle, showLocation, showFavicon bool) (bool, string, string, string, string, string) {
    client := &http.Client{
        Timeout: time.Second * 5,
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            if !followRedirects {
                return http.ErrUseLastResponse
            }
            if len(via) >= maxRedirects {
                return fmt.Errorf("stopped after %d redirects", maxRedirects)
            }
            if !followHostRedirects {
                u, err := url.Parse(req.URL.String())
                if err != nil {
                    return err
                }
                for _, prevReq := range via {
                    prevURL, err := url.Parse(prevReq.URL.String())
                    if err != nil {
                        return err
                    }
                    if u.Host != prevURL.Host {
                        return http.ErrUseLastResponse
                    }
                }
            }
            return nil
        },
    }

    resp, err := client.Get("http://" + domain)
    if err != nil {
        return false, "", "", "", "", ""
    }
    defer resp.Body.Close()

    statusCode := fmt.Sprintf("%d", resp.StatusCode)
    if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound {
        ip, err := getIP(domain)
        if err != nil {
            return true, "N/A", statusCode, "", "", ""
        }
        title := ""
        if showTitle {
            title = getTitle(domain)
        }
        location := ""
        if showLocation {
            location = resp.Header.Get("Location")
        }
        faviconHash := ""
        if showFavicon {
            faviconHash = getFaviconHash(domain)
        }
        return true, ip, statusCode, title, location, faviconHash
    }
    return false, "", statusCode, "", "", ""
}

// Function to get IP address of a domain
func getIP(domain string) (string, error) {
    ips, err := net.LookupIP(domain)
    if err != nil {
        return "", err
    }
    return ips[0].String(), nil
}

// Function to read lines from a file
func readLines(filename string) ([]string, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var lines []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        lines = append(lines, strings.TrimSpace(scanner.Text()))
    }
    return lines, scanner.Err()
}

// Function to get the title of a webpage
func getTitle(url string) string {
    resp, err := http.Get("http://" + url)
    if err != nil {
        return ""
    }
    defer resp.Body.Close()

    doc, err := goquery.NewDocumentFromReader(resp.Body)
    if err != nil {
        return ""
    }

    title := doc.Find("title").Text()
    title = strings.TrimSpace(title)
    if title == "" {
        return ""
    }
    return title
}

// Function to calculate mmh3 hash for '/favicon.ico' file
func getFaviconHash(domain string) string {
    resp, err := http.Get("http://" + domain + "/favicon.ico")
    if err != nil {
        return "N/A"
    }
    defer resp.Body.Close()

    hash := fnv.New64a()
    if _, err := hash.Write([]byte("/favicon.ico")); err != nil {
        return "N/A"
    }

    hashValue := murmur3.New64()
    if _, err := hashValue.Write([]byte("/favicon.ico")); err != nil {
        return "N/A"
    }

    return fmt.Sprintf("%x", hash.Sum64())
}
