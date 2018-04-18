package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	auditapi "k8s.io/apiserver/pkg/apis/audit/v1beta1"

	_ "github.com/mattn/go-sqlite3"
)

/*

Sample queries:

# load db:
sqlite3 <path to db>

SELECT SUM(count),user,verb,namespace,apigroup,resource FROM audit GROUP BY user,verb,namespace,apigroup,resource ORDER BY count DESC LIMIT 20;

# Get the top request users, with nodes aggregated (just clipping the node-specific call)
SELECT SUM(count) AS total, SUBSTR(user,0,44) as usr FROM audit GROUP BY usr ORDER BY total DESC LIMIT 50;

*/

var (
	logFile = flag.String("logs", "", "Path to audit logs")
	dbFile  = flag.String("db", "", "Path to write DB to")
	logType = flag.String("log-type", "json", "Log file format (json or legacy)")
)

const (
	// Name of the table holding the results.
	Table = "audit"
)

type event struct {
	IP          string
	User        string
	Verb        string
	Namespace   string
	Group       string
	Resource    string
	Subresource string
	Name        string
	URI         string
	Response    string
}

const schema = `
count BIGINT NOT NULL,
ip VARCHAR(64) NOT NULL,
user VARCHAR(64) NOT NULL,
verb VARCHAR(16) NOT NULL,
namespace VARCHAR(128),
apigroup VARCHAR(64),
resource VARCHAR(64),
subresource VARCHAR(64),
name VARCHAR(128),
uri TEXT,
response VARCHAR(16)
`

func main() {
	flag.Parse()

	if *logFile == "" {
		log.Fatalf("required --log-file not specified")
	}
	if *dbFile == "" {
		log.Fatalf("required --db not specified")
	}

	logReader, err := os.Open(*logFile)
	if err != nil {
		log.Fatalf("Failed to read logs: %v", err)
	}
	defer logReader.Close()

	summary, err := summarize(logReader)
	if err != nil {
		log.Fatalf("Failed to read logs: %v", err)
	}

	db, err := sql.Open("sqlite3", *dbFile)
	if err != nil {
		log.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	// Setup pragmas.
	pragmas := map[string]string{
		"synchronous":   "OFF",
		"count_changes": "OFF",
		"journal_mode":  "MEMORY",
		"temp_store":    "MEMORY",
	}
	for pragma, val := range pragmas {
		if _, err := db.Exec(fmt.Sprintf("PRAGMA %s=%s", pragma, val)); err != nil {
			log.Fatalf("PRAGMA Error: %v", err)
		}
	}

	// Create table if necessary.
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", Table, schema)
	_, err = db.Exec(query)
	if err != nil {
		log.Fatalf("CREATE TABLE Error: %v\n%s", err, query)
	}

	insert, err := db.Prepare("INSERT INTO audit (" +
		"count, ip, user, verb, namespace, apigroup, resource, subresource, name, uri, response" +
		") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatalf("PREPARE Error: %v\n", err)
	}
	for ev, count := range summary {
		_, err := insert.Exec(count, ev.IP, ev.User, ev.Verb, ev.Namespace, ev.Group, ev.Resource,
			ev.Subresource, ev.Name, ev.URI, ev.Response)
		if err != nil {
			log.Fatalf("INSERT Error: %v\n", err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

func summarize(logs io.Reader) (map[event]int, error) {
	summary := map[event]int{}

	read := 0
	parsed := 0
	defer func() {
		log.Printf("Read %d lines. Parsed %d lines. Counted %d unique events.", read, parsed, len(summary))
	}()

	scanner := bufio.NewScanner(logs)
	for scanner.Scan() {
		line := scanner.Text()
		read++

		ev, err := parseAuditLine(line)
		if err != nil {
			return nil, err
		}

		if ev == nil {
			continue
		}
		parsed++

		summary[*ev] = summary[*ev] + 1
	}
	return summary, scanner.Err()
}

func parseAuditLine(line string) (*event, error) {
	switch *logType {
	case "json":
		return parseJsonAuditLine(line)
	case "legacy":
		return parseLegacyAuditLine(line)
	default:
		return nil, fmt.Errorf("invalid log type: %s", *logType)
	}
}

func parseJsonAuditLine(line string) (*event, error) {
	var evt auditapi.Event
	if err := json.Unmarshal([]byte(line), &evt); err != nil {
		return nil, err
	}

	sum := &event{
		User: evt.User.Username,
		Verb: evt.Verb,
		URI:  evt.RequestURI,
	}
	if obj := evt.ObjectRef; obj != nil {
		sum.Group = obj.APIGroup
		sum.Resource = obj.Resource
		sum.Subresource = obj.Subresource
		sum.Name = obj.Name
		sum.Namespace = obj.Namespace
	}
	if len(evt.SourceIPs) > 0 {
		sum.IP = evt.SourceIPs[len(evt.SourceIPs)-1]
	}
	return sum, nil
}

func parseLegacyAuditLine(line string) (*event, error) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nil, fmt.Errorf("could not parse audit line: %s", line)
	}
	// Ignore first 2 fields (<timestamp> AUDIT:)
	fields = fields[2:]
	ev := event{}
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
			ev.User = value
		case "method":
			ev.Verb = value
		case "namespace":
			ev.Namespace = value
		case "ip":
			ev.IP = value
		case "response":
			ev.Response = value
		case "uri":
			parts := strings.Split(strings.Trim(value, "/"), "/")
			if len(parts) < 3 {
				ev.URI = value
				continue
			}
			namespaced := ev.Namespace != "<none>" && ev.Namespace != ""
			if parts[0] == "api" {
				ev.Group = "core"
				if namespaced && len(parts) > 4 {
					ev.Resource = parts[4]
					if len(parts) > 5 {
						ev.Name = parts[5]
					}
					if len(parts) > 6 {
						ev.Subresource = parts[6]
					}
				} else if len(parts) > 2 {
					ev.Resource = parts[2]
					if len(parts) > 3 {
						ev.Name = parts[3]
					}
					if len(parts) > 4 {
						ev.Subresource = parts[4]
					}
				} else {
					ev.URI = value
				}
			} else if parts[0] == "apis" {
				ev.Group = parts[1]
				if namespaced && len(parts) > 5 {
					ev.Resource = parts[5]
					if len(parts) > 6 {
						ev.Name = parts[6]
					}
					if len(parts) > 7 {
						ev.Subresource = parts[7]
					}
				} else if len(parts) > 3 {
					ev.Resource = parts[2]
					if len(parts) > 4 {
						ev.Name = parts[4]
					}
					if len(parts) > 5 {
						ev.Subresource = parts[5]
					}
				} else {
					ev.URI = value
				}
			} else {
				ev.URI = value
			}
		case "stage":
			if value != "ResponseComplete" {
				return nil, nil
			}
		}
	}
	return &ev, nil
}
