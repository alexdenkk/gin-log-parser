package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"encoding/json"
)

// Struct of log record
type LogRecord struct {
	Date     time.Time     `json:"date"`
	Code     int           `json:"code"`
	Duration time.Duration `json:"duration"`
	IP       string        `json:"ip"`
	Method   string        `json:"method"`
	URL      string        `json:"url"`
}

// Struct of metrics
type Metrics struct {
	Count        int `json:"count"`
	TotalTime    time.Duration
	MinTime      time.Duration
	MaxTime      time.Duration
	StatusCounts map[int]int
}

func main() {
	// Filters
	var method, date, url, ip string
	var code int

	// Output modes
	var raw bool
	var json bool

	// Flag parsing
	flag.StringVar(&method, "method", "", "HTTP method to filter")
	flag.IntVar(&code, "code", 0, "Status code to filter")
	flag.StringVar(&date, "date", "", "Date to filter (format: YYYY/MM/DD)")
	flag.StringVar(&url, "url", "", "URL path to filter")
	flag.StringVar(&ip, "ip", "", "IP address to filter")
	flag.BoolVar(&raw, "raw", false, "Output filtered logs instead of statistics")
	flag.BoolVar(&json, "json", false, "Output logs in JSON format")
	flag.Parse()

	// Scanning input and parsing logs
	scanner := bufio.NewScanner(os.Stdin)
	var records []LogRecord

	for scanner.Scan() {
		line := scanner.Text()
		record, err := parseLine(line)
		if err != nil {
			continue
		}

		if !matchesFilter(record, method, code, date, url, ip) {
			continue
		}

		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	// Output
	if json {
		printJSON(records)
		os.Exit(0)
	}

	if raw {
		printRaw(records)
		os.Exit(0)
	}

	metrics := calculateMetrics(records)
	printMetrics(metrics)
}

// Line parsing
func parseLine(line string) (LogRecord, error) {
	if !strings.HasPrefix(line, "[GIN]") {
		return LogRecord{}, fmt.Errorf("invalid format")
	}

	parts := strings.Split(line, "|")
	if len(parts) != 5 {
		return LogRecord{}, fmt.Errorf("invalid format")
	}

	datePart := strings.TrimSpace(parts[0])
	codePart := strings.TrimSpace(parts[1])
	durationPart := strings.TrimSpace(parts[2])
	ipPart := strings.TrimSpace(parts[3])
	methodUrlPart := strings.TrimSpace(parts[4])

	dateStr := strings.Fields(datePart)[1] + " " + strings.Fields(datePart)[3]
	parsedDate, err := time.Parse("2006/01/02 15:04:05", dateStr)
	if err != nil {
		return LogRecord{}, err
	}

	parsedCode, err := strconv.Atoi(codePart)
	if err != nil {
		return LogRecord{}, err
	}

	parsedDuration, err := parseDuration(durationPart)
	if err != nil {
		return LogRecord{}, err
	}

	methodUrlParts := strings.Fields(methodUrlPart)
	if len(methodUrlParts) < 2 {
		return LogRecord{}, fmt.Errorf("invalid method/URL format")
	}

	return LogRecord{
		Date:     parsedDate,
		Code:     parsedCode,
		Duration: parsedDuration,
		IP:       ipPart,
		Method:   methodUrlParts[0],
		URL:      strings.Join(methodUrlParts[1:], " "),
	}, nil
}

// Duration parsing
func parseDuration(durStr string) (time.Duration, error) {
	durStr = strings.TrimSpace(durStr)
	
	if strings.HasSuffix(durStr, "µs") {
		val, err := strconv.ParseFloat(strings.TrimSuffix(durStr, "µs"), 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(val * float64(time.Microsecond)), nil
	}

	if strings.HasSuffix(durStr, "ms") {
		val, err := strconv.ParseFloat(strings.TrimSuffix(durStr, "ms"), 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(val * float64(time.Millisecond)), nil
	}

	return time.ParseDuration(durStr)
}

// Checking is line matching GIN logs format
func matchesFilter(record LogRecord, method string, code int, date string, url string, ip string) bool {
	if method != "" && record.Method != method {
		return false
	}
	
	if code != 0 && record.Code != code {
		return false
	}
	
	if date != "" && record.Date.Format("2006/01/02") != date {
		return false
	}
	
	if url != "" && record.URL != url {
		return false
	}
	
	if ip != "" && record.IP != ip {
		return false
	}
	
	return true
}

// Calculation of metrics
func calculateMetrics(records []LogRecord) Metrics {
	if len(records) == 0 {
		return Metrics{StatusCounts: make(map[int]int)}
	}

	metrics := Metrics{
		MinTime:      records[0].Duration,
		MaxTime:      records[0].Duration,
		StatusCounts: make(map[int]int),
	}

	for _, record := range records {
		metrics.Count++
		metrics.TotalTime += record.Duration
		metrics.StatusCounts[record.Code]++

		if record.Duration < metrics.MinTime {
			metrics.MinTime = record.Duration
		}
		if record.Duration > metrics.MaxTime {
			metrics.MaxTime = record.Duration
		}
	}

	return metrics
}

// JSON mode output
func printJSON(records []LogRecord) {
	formatted, err := json.Marshal(records)

	if err != nil {
		fmt.Errorf("failed to encode in json: %w", err)
		os.Exit(1)
	}

	fmt.Println(string(formatted))
}

// Metrics mode output
func printMetrics(metrics Metrics) {
	fmt.Printf("Total Requests: %d\n", metrics.Count)
	
	if metrics.Count == 0 {
		return
	}

	fmt.Printf("Total Time: %v\n", metrics.TotalTime)
	fmt.Printf("Average Time: %v\n", metrics.TotalTime/time.Duration(metrics.Count))
	fmt.Printf("Min Time: %v\n", metrics.MinTime)
	fmt.Printf("Max Time: %v\n", metrics.MaxTime)
	fmt.Println("\nStatus Code Distribution:")
	
	for code, count := range metrics.StatusCounts {
		fmt.Printf("  %d: %d\n", code, count)
	}
}

// Raw mode output
func printRaw(records []LogRecord) {
	for _, record := range records {
		fmt.Printf("%s | %3d | %12s | %15s | %-7s %s\n",
			record.Date.Format("2006/01/02 - 15:04:05"),
			record.Code,
			strings.TrimSpace(formatDuration(record.Duration)),
			strings.TrimSpace(record.IP),
			strings.TrimSpace(record.Method),
			strings.TrimSpace(record.URL),
		)
	}
}

// Duration formatting
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%.3fns", float64(d.Nanoseconds()))

	} else if d < time.Millisecond {
		return fmt.Sprintf("%.3fµs", float64(d.Microseconds()))

	} else if d < time.Second {
		return fmt.Sprintf("%.3fms", float64(d.Milliseconds()))
	}

	return fmt.Sprintf("%.3fs", d.Seconds())
}
