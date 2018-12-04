package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

var (
	godeps = flag.String("godep-path", "~/go/src/k8s.io/kubernetes/Godeps/Godeps.json",
		"Path to the Godeps.json file to scan")
	sortField = flag.String("sort", "repo", "field to sort by, append ! to sort descending")
	workdir   = flag.String("working-dir", "", "directory to stage git repo in. Defaults to a tmpdir.")
)

func main() {
	flag.Parse()
	sortFunc(nil) // Validate flag

	deps := readGodeps()
	results := scanDeps(deps)
	printResults(results)
}

type DepResult struct {
	Repo           string
	Age            string
	LatestChange   string
	NumCommitsDiff int
}

func sortFunc(results []DepResult) func(i, j int) bool {
	descending := false
	if strings.HasSuffix(*sortField, "!") {
		descending = true
		*sortField = strings.TrimSuffix(*sortField, "!")
	}
	switch strings.ToLower(*sortField) {
	case "repo":
		return func(i, j int) bool {
			return (results[i].Repo < results[j].Repo) != descending
		}
	case "commits":
		return func(i, j int) bool {
			return (results[i].NumCommitsDiff < results[j].NumCommitsDiff) != descending
		}
	}
	log.Fatalf("Unknown sort format: %s", *sortField)
	return func(i, j int) bool { return false }
}

func printResults(results []DepResult) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "REPO\tAGE\tLATEST\tDIFF")
	sort.Slice(results, sortFunc(results))
	for _, result := range results {
		fmt.Fprintf(w, "%s,\t%s,\t%s,\t%d\n", result.Repo, result.Age, result.LatestChange, result.NumCommitsDiff)
	}
	w.Flush()
}

func scanDeps(deps map[string]string) []DepResult {
	if *workdir == "" {
		tmp, err := ioutil.TempDir("", "")
		if err != nil {
			log.Fatalf("Failed to create temp dir: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tmp); err != nil {
				log.Printf("WARNING: Failed to remove temporary directory %s: %v", tmp, err)
			}
		}()
		*workdir = tmp
	}

	if err := os.Chdir(*workdir); err != nil {
		log.Fatalf("Failed to chdir %s: %v", *workdir, err)
	}

	// Initialize temp git repo
	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to initialize temp git repo: %v", err)
	}

	repos := map[string]DepResult{}
	i := 0
	for repo, rev := range deps {
		log.Printf("Scanning %s [%d of %d]", repo, i, len(deps))
		repos[repo] = scanDep(repo, rev, *workdir)
		i++
	}

	results := make([]DepResult, 0, len(repos))
	for repo, res := range repos {
		res.Repo = repo
		results = append(results, res)
	}
	return results
}

func scanDep(repo, rev, workingDir string) DepResult {
	result := DepResult{}

	// 1. Fetch remote
	remote := fmt.Sprintf("https://%s.git", repo)
	if err := exec.Command("git", "fetch", remote, "master").Run(); err != nil {
		log.Printf("WARNING: Failed to fetch %s: %v", remote, err)
		return result
	}

	// 2. Lookup commit date
	if date, err := exec.Command("git", "show", "--quiet", "--format=format:%cr", rev).Output(); err != nil {
		log.Printf("WARNING: Failed to lookup commit date: %v", err)
		result.Age = "ERR"
	} else {
		result.Age = strings.Replace(string(date), ",", "", -1)
	}

	// 3. Lookup latest commit date.
	if date, err := exec.Command("git", "show", "--quiet", "--format=format:%cr", "FETCH_HEAD").Output(); err != nil {
		log.Printf("WARNING: Failed to lookup FETCH_HEAD commit date: %v", err)
		result.LatestChange = "ERR"
	} else {
		result.LatestChange = strings.Replace(string(date), ",", "", -1)
	}

	// 4. Count commits since rev.
	commits, err := exec.Command("git", "rev-list", "--count", fmt.Sprintf("%s..FETCH_HEAD", rev)).Output()
	if err != nil {
		log.Printf("WARNING: Failed to count commits: %v", err)
	}
	result.NumCommitsDiff, err = strconv.Atoi(strings.TrimSpace(string(commits)))
	if err != nil {
		log.Printf("WARNING: Failed to parse commit count %s: %v", commits, err)
	}

	return result
}

func runCmd(cmd *exec.Cmd, workingDir, action string) string {
	cmd.Dir = workingDir
	out, err := cmd.Output()
	if err != nil {
		log.Printf("WARNING: Failed to %s: %v", action, err)
		return ""
	}
	return string(out)
}

func readGodeps() map[string]string {
	path := expandPath(*godeps)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read file %q: %v", path, err)
	}

	deps := &Godeps{}
	err = json.Unmarshal(data, deps)
	if err != nil {
		log.Fatalf("Failed to unmarshal Godeps: %v", err)
	}

	results := map[string]string{}
	for _, dep := range deps.Deps {
		if !strings.HasPrefix(dep.ImportPath, "github.com") {
			log.Printf("  Skipping non-github import: %s", dep.ImportPath)
			continue
		}

		parts := strings.Split(dep.ImportPath, "/")
		if len(parts) < 3 {
			log.Printf("WARNING: Unknown import format: %s", dep.ImportPath)
			continue
		}

		repo := strings.Join(parts[:3], "/")
		if rev, ok := results[repo]; ok && rev != dep.Rev {
			log.Printf("WARNING: Conflicting rev: %s", dep.ImportPath)
			continue
		}
		results[repo] = dep.Rev
	}
	return results
}

func expandPath(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Cannot read current user: %v", err)
	}
	if path == "~" {
		return usr.HomeDir
	}
	return filepath.Join(usr.HomeDir, path[2:])
}

type Godeps struct {
	ImportPath   string
	GoVersion    string
	GodepVersion string
	Packages     []string `json:",omitempty"` // Arguments to save, if any.
	Deps         []Dependency
}

type Dependency struct {
	ImportPath string
	Comment    string `json:",omitempty"` // Description of commit, if present.
	Rev        string // VCS-specific commit ID.
}
