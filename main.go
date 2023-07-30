package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

func main() {
	filePath := flag.String("file", "", "The path to the CSV file")
	numThreads := flag.Int("threads", 1, "Number of threads in the processing")
	outputFile := flag.String("o", "", "Output file name")
	subdomainFlag := flag.Bool("sf", false, "Indicates whether the CSV file contains subdomains")
	jsonOutputFlag := flag.Bool("json", false, "Save results in JSON format")
	flag.Parse()

	if *subdomainFlag && *filePath == "" {
		fmt.Println("Error: the path to the CSV file is required when using the -sf flag")
		return
	}

	if *numThreads <= 0 {
		fmt.Println("Error: the number of threads must be greater than 0")
		return
	}

	var records [][]string
	if *subdomainFlag {
		var err error
		records, err = readSubdomainCSV(*filePath)
		if err != nil {
			fmt.Println("Error reading subdomain CSV file:", err)
			return
		}
	} else {
		file, err := os.Open(*filePath)
		if err != nil {
			fmt.Println("Error when opening the file:", err)
			return
		}
		defer file.Close()

		reader := csv.NewReader(file)
		records, err = reader.ReadAll()
		if err != nil {
			fmt.Println("Error when reading CSV file:", err)
			return
		}
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
				ips := getIPsFromRecord(domain, *subdomainFlag, records)
				if len(ips) == 0 {
					results <- fmt.Sprintf("%s; N/A; N/A; N/A; N/A", domain)
					continue
				}

				for _, ip := range ips {
					headers, err := fetchHeaders(domain, false)
					if err != nil {
						headers, err = fetchHeaders(domain, true)
						if err != nil {
							results <- fmt.Sprintf("%s; %s; N/A; N/A; N/A", domain, ip)
							continue
						}
					}

					body, err := fetchResponseBody(domain, false)
					if err != nil {
						body, err = fetchResponseBody(domain, true)
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

	if *jsonOutputFlag {
		err := saveResultsToJSON(*outputFile, results)
		if err != nil {
			fmt.Println("Error saving results to JSON file:", err)
		} else {
			fmt.Println("Results saved to", *outputFile)
		}
	} else if *outputFile == "" {
		printResultsToConsole(results)
	} else {
		err := saveResultsToCSV(*outputFile, results)
		if err != nil {
			fmt.Println("Error saving results to CSV file:", err)
		} else {
			fmt.Println("Results saved to", *outputFile)
		}
	}
}

func printResultsToConsole(results chan string) {
	fmt.Println("Domain; IP Addr; Header; WP; Redirect")
	for result := range results {
		fmt.Println(result)
	}
}

func saveResultsToCSV(outputFile string, results chan string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	outputWriter := csv.NewWriter(file)
	defer outputWriter.Flush()

	headers := []string{"Domain", "IP Addr", "Header", "WP", "Redirect"}
	outputWriter.Write(headers)

	for result := range results {
		data := strings.Split(result, "; ")
		outputWriter.Write(data)
	}

	return nil
}

func saveResultsToJSON(outputFile string, results chan string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var jsonData []map[string]string

	for result := range results {
		data := strings.Split(result, "; ")
		record := map[string]string{
			"Domain":   data[0],
			"IP Addr":  data[1],
			"Header":   data[2],
			"WP":       data[3],
			"Redirect": data[4],
		}
		jsonData = append(jsonData, record)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(jsonData); err != nil {
		return err
	}

	return nil
}

func readSubdomainCSV(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	var records [][]string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

func getIPsFromRecord(domain string, subdomainFlag bool, records [][]string) []string {
	var ips []string

	if subdomainFlag {
		for _, record := range records {
			subdomain := record[0]
			ip := record[1]
			isCloudflare := record[2]

			if subdomain == domain && !strings.EqualFold(isCloudflare, "true") {
				ips = append(ips, ip)
			}
		}
	} else {
		for _, record := range records {
			recordDomain := record[0]
			if recordDomain == domain {
				ip := record[1]
				ips = append(ips, ip)
			}
		}
	}

	return ips
}

func fetchHeaders(domain string, useHTTPS bool) (http.Header, error) {
	scheme := "http"
	if useHTTPS {
		scheme = "https"
	}

	url := fmt.Sprintf("%s://%s", scheme, domain)
	response, err := http.Head(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return response.Header, nil
}

func fetchResponseBody(domain string, useHTTPS bool) ([]byte, error) {
	scheme := "http"
	if useHTTPS {
		scheme = "https"
	}

	response, err := http.Get(fmt.Sprintf("%s://%s", scheme, domain))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
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
