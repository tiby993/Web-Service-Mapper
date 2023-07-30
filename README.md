# Web-Services-Mapper

"web-service-mapper" is a Go program that maps various details of web services from a CSV file containing domain names. It can process the data concurrently using multiple threads specified by the user. The program resolves the IP addresses of the domains and fetches HTTP headers and response bodies from the ports 80 (HTTP) and 443 (HTTPS) to determine if the web service is a WordPress site or if it performs redirects. The results can be displayed on the console, saved in CSV format, or exported to a JSON file based on user preferences. Additionally, the program supports subdomain CSV files, enabling it to process subdomains and their associated IP addresses without DNS resolution.


## Usage

```
./main -file domains.csv
```
## Flags

-file / --file: This switch is used to specify the path to the CSV file containing the domain names. It is a required switch, and without it, the program will not know which CSV file to process.

Example usage:
```
./main -file domains.csv
```

-threads / --threads: This switch allows you to set the number of threads the program should use for concurrent processing. The program will process the domains concurrently to improve performance. The default value is 1 if not specified.

Example usage:
```
./main -file domains.csv -threads 4
```

-o / --output: This switch is used to specify the output file name or path where the results will be saved. It can be used for both CSV and JSON formats.

Example usage (saving results in CSV format):
```
./main -file domains.csv -o output.csv
```

Example usage (saving results in JSON format):
```
./main -file domains.csv -json -o output.json
```

-sf / --sf: This switch indicates that the CSV file contains subdomains instead of just domain names. It allows the program to process subdomain CSV files correctly.

Example usage (with a subdomain CSV file):
```
./main -sf -file subdomains.csv
```

-json / --json: This switch is used to specify that the results should be saved in JSON format. It can be combined with the -o switch to specify the output file name.

Example usage:
```
./main -file domains.csv -json -o output.json
```
