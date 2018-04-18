package main

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	auditapi "k8s.io/apiserver/pkg/apis/audit/v1beta1"
)

var (
	policyFile   = flag.String("policy-file", "", "Path to policy file to apply")
	logFile      = flag.String("log-file", "", "Path to audit logs")
	logType      = flag.String("log-type", "json", "Log file format (json or legacy)")
	reportLength = flag.Int("n", 0, "Number of lines to include in the report")
	cacheFile    = flag.String("cache-file", "", "Path to summary cache. "+
		"If a logFile is included, the cache is written. Otherwise, it is read.")
	aggregate = flag.String("agg", "", "Comma-separated list of columns to ignore, so events are aggregated across all values.")
	ignore    = flag.String("ignore", "", "Comma-separated list of key=value pairs identifying rows to ignore. The key is the column and the value is the value to filter.")
)

type summary struct {
	User, Verb, Namespace, Group, Resource, Subresource, Name string
	URI                                                       string
}

func main() {
	flag.Parse()

	if *policyFile != "" {
		log.Fatalf("NYI: policy-file")
	}

	sum := map[summary]int{}
	if *logFile != "" {
		logReader, err := os.Open(*logFile)
		if err != nil {
			log.Fatalf("Failed to read logs: %v", err)
		}
		defer logReader.Close()

		sum, err = summarize(logReader)
		if err != nil {
			log.Fatalf("Failed to summarize logs: %v", err)
		}
		log.Printf("Summarized %d unique events.", len(sum))

		if *cacheFile != "" {
			sumWriter, err := os.OpenFile(*cacheFile, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				log.Printf("Failed to open summary cache for writing: %v", err)
			}
			defer sumWriter.Close()
			enc := gob.NewEncoder(sumWriter)
			err = enc.Encode(sum)
			if err != nil {
				log.Printf("Failed to write summary cache: %v", err)
			}
		}
	} else if *cacheFile != "" {
		sumReader, err := os.Open(*cacheFile)
		if err != nil {
			log.Fatalf("Failed to open summary cache for reading: %v", err)
		}
		defer sumReader.Close()

		dec := gob.NewDecoder(sumReader)
		err = dec.Decode(&sum)
		if err != nil {
			log.Fatalf("Failed to read summary cache: %v", err)
		}
		log.Printf("Read %d summarized events.", len(sum))
	} else {
		log.Fatalf("Must supply either --policy-file or --cache-file")
	}

	// Aggregate summaries based on ignored columns.
	aggSet := map[string]bool{}
	if *aggregate != "" {
		for _, col := range strings.Split(*aggregate, ",") {
			aggSet[col] = true
		}
		agg := map[summary]int{}
		for s, count := range sum {
			for col := range aggSet {
				switch col {
				case "user":
					s.User = ""
				case "verb":
					s.Verb = ""
				case "namespace":
					s.Namespace = ""
				case "group":
					s.Group = ""
				case "resource":
					s.Resource = ""
				case "uri":
					s.URI = ""
				}
			}
			agg[s] = agg[s] + count
		}
		sum = agg
	}

	// Ignore rows as necessary.
	if *ignore != "" {
		ignored := map[string]map[string]bool{}
		for _, ig := range strings.Split(*ignore, ",") {
			parts := strings.SplitN(ig, "=", 2)
			if len(parts) != 2 {
				log.Printf("WARNING: invalid ignore flag: %s", ig)
				continue
			}
			aggSet[parts[0]][parts[1]]
		}
		agg := map[summary]int{}
		for s, count := range sum {
			for col := range aggSet {
				switch col {
				case "user":
					s.User = ""
				case "verb":
					s.Verb = ""
				case "namespace":
					s.Namespace = ""
				case "group":
					s.Group = ""
				case "resource":
					s.Resource = ""
				case "uri":
					s.URI = ""
				}
			}
			agg[s] = agg[s] + count
		}
		sum = agg
	}

	fmt.Println()
	tab := tabwriter.NewWriter(os.Stdout, 4, 2, 2, ' ', tabwriter.DiscardEmptyColumns)
	// Print header:
	fmt.Fprintf(tab, "Count")
	separator := "-----"
	for _, col := range []string{"User", "Verb", "Namespace", "Group", "Resource", "Subresource", "Name", "URI"} {
		if aggSet[strings.ToLower(col)] {
			col = ""
		}
		fmt.Fprintf(tab, "\t"+col)
		separator = separator + "\t" + strings.Repeat("-", len(col))
	}
	fmt.Fprintf(tab, "\n%s\n", separator)

	for i, sum := range sortSummary(sum) {
		if *reportLength != 0 && i >= *reportLength {
			break
		}
		s := sum.sum
		fmt.Fprintf(tab, "%6d\t%s\n", sum.count, strings.Join(
			[]string{s.User, s.Verb, s.Namespace, s.Group, s.Resource, s.Subresource, s.Name, s.URI}, "\t"))
	}
	if err := tab.Flush(); err != nil {
		log.Printf("Output error: %v", err)
	}
}

type summaryCount struct {
	sum   summary
	count int
}

func sortSummary(sum map[summary]int) []summaryCount {
	sorted := make([]summaryCount, 0, len(sum))
	for s, count := range sum {
		sorted = append(sorted, summaryCount{s, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		// Sort descending.
		return sorted[i].count > sorted[j].count
	})
	return sorted
}

func summarize(logs io.Reader) (map[summary]int, error) {
	sum := map[summary]int{}

	read := 0
	parsed := 0
	defer func() {
		log.Printf("Read %d lines. Parsed %d lines.", read, parsed)
	}()

	scanner := bufio.NewScanner(logs)
	for scanner.Scan() {
		line := scanner.Text()
		read++

		s, err := parseAuditLine(line)
		if err != nil {
			return nil, err
		}

		if s == nil {
			continue
		}
		parsed++

		sum[*s] = sum[*s] + 1
	}
	return sum, scanner.Err()
}

func parseAuditLine(line string) (*summary, error) {
	switch *logType {
	case "json":
		return parseJsonAuditLine(line)
	case "legacy":
		return parseLegacyAuditLine(line)
	default:
		return nil, fmt.Errorf("invalid log type: %s", *logType)
	}
}

func parseJsonAuditLine(line string) (*summary, error) {
	var event auditapi.Event
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return nil, err
	}

	sum := &summary{
		User: event.User.Username,
		Verb: event.Verb,
		URI:  event.RequestURI,
	}
	if obj := event.ObjectRef; obj != nil {
		sum.Group = obj.APIGroup
		sum.Resource = obj.Resource
		sum.Subresource = obj.Subresource
		sum.Name = obj.Name
		sum.Namespace = obj.Namespace
	}
	return sum, nil
}

func parseLegacyAuditLine(line string) (*summary, error) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nil, fmt.Errorf("could not parse audit line: %s", line)
	}
	// Ignore first 2 fields (<timestamp> AUDIT:)
	fields = fields[2:]
	sum := summary{}
	for _, f := range fields {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("could not parse audit line (part: %q): %s", f, line)
		}
		value := strings.Trim(parts[1], "\"")
		if strings.ContainsRune(value, '?') {
			value = strings.SplitN(value, "?", 2)[0]
		}
		switch parts[0] {
		case "user":
			sum.User = value
		case "method":
			sum.Verb = value
		case "namespace":
			sum.Namespace = value
		case "uri":
			parts := strings.Split(strings.Trim(value, "/"), "/")
			if len(parts) < 3 {
				sum.URI = value
				continue
			}
			namespaced := sum.Namespace != "<none>" && sum.Namespace != ""
			if parts[0] == "api" {
				sum.Group = "core"
				if namespaced && len(parts) > 4 {
					sum.Resource = parts[4]
					if len(parts) > 5 {
						sum.Name = parts[5]
					}
					if len(parts) > 6 {
						sum.Subresource = parts[6]
					}
				} else if len(parts) > 2 {
					sum.Resource = parts[2]
					if len(parts) > 3 {
						sum.Name = parts[3]
					}
					if len(parts) > 4 {
						sum.Subresource = parts[4]
					}
				} else {
					sum.URI = value
				}
			} else if parts[0] == "apis" {
				sum.Group = parts[1]
				if namespaced && len(parts) > 5 {
					sum.Resource = parts[5]
					if len(parts) > 6 {
						sum.Name = parts[6]
					}
					if len(parts) > 7 {
						sum.Subresource = parts[7]
					}
				} else if len(parts) > 3 {
					sum.Resource = parts[2]
					if len(parts) > 4 {
						sum.Name = parts[4]
					}
					if len(parts) > 5 {
						sum.Subresource = parts[5]
					}
				} else {
					sum.URI = value
				}
			} else {
				sum.URI = value
			}
		case "stage":
			if value != "ResponseComplete" {
				return nil, nil
			}
		}
	}
	return &sum, nil
}
