package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

func main() {
	filePath := flag.String("file", "domains.csv", "The path to the domains CSV file")
	numThreads := flag.Int("threads", 1, "Number of threads in the processing")
	flag.Parse()

	if *numThreads <= 0 {
		fmt.Println("Error: the number of threads must be greater than 0")
		return
	}

	file, err := os.Open(*filePath)
	if err != nil {
		fmt.Println("Error when opening the file:", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error when reading CSV file:", err)
		return
	}

	domains := make(chan string, len(records))
	results := make(chan string, len(records))

	for _, record := range records {
		domain := record[0]
		domains <- domain
	}
	close(domains)

	var wg sync.WaitGroup
	wg.Add(*numThreads)

	// Starting the threads
	for i := 0; i < *numThreads; i++ {
		go func() {
			defer wg.Done()

			for domain := range domains {
				ips, err := resolveIP(domain)
				if err != nil {
					results <- fmt.Sprintf("%s; N/A; N/A; N/A; N/A", domain)
					continue
				}

				for _, ip := range ips {
					headers, err := fetchHeaders(domain, 80)
					if err != nil {
						headers, err = fetchHeaders(domain, 443)
						if err != nil {
							results <- fmt.Sprintf("%s; %s; N/A; N/A; N/A", domain, ip)
							continue
						}
					}

					body, err := fetchResponseBody(domain, 80)
					if err != nil {
						body, err = fetchResponseBody(domain, 443)
						if err != nil {
							results <- fmt.Sprintf("%s; %s; %s; N/A; N/A", domain, ip, headers.Get("Server"))
							continue
						}
					}

					wp := "No"
					if isWordPress(body) {
						wp = "Yes"
					}

					redirectURL := headers.Get("Location")
					redirect := "No"
					if redirectURL != "" {
						redirect = redirectURL
					}

					results <- fmt.Sprintf("%s; %s; %s; %s; %s", domain, ip, headers.Get("Server"), wp, redirect)
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	fmt.Println("Domain; IP Addr; Header; WP; Redirect")
	for result := range results {
		fmt.Println(result)
	}
}

func resolveIP(domain string) ([]string, error) {
	ips, err := net.LookupHost(domain)
	if err != nil {
		return nil, err
	}
	return ips, nil
}

func fetchHeaders(domain string, port int) (http.Header, error) {
	response, err := http.Head(fmt.Sprintf("http://%s:%d", domain, port))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return response.Header, nil
}

func fetchResponseBody(domain string, port int) ([]byte, error) {
	response, err := http.Get(fmt.Sprintf("http://%s:%d", domain, port))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return ioutil.ReadAll(response.Body)
}

func isWordPress(body []byte) bool {
	bodyStr := string(body)
	keywords := []string{"wp-content", "wp-includes", "wp-json"}
	for _, keyword := range keywords {
		if strings.Contains(bodyStr, keyword) {
			return true
		}
	}
	return false
}
