package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

const (
	profilePath    = "/profile"
	profileNameVar = "%%PROFILE%%"
	profile        = `
#include <tunables/global>

profile ` + profileNameVar + ` flags=(attach_disconnected,mediate_deleted) {

  #include <abstractions/base>

  network,
  capability,
  file,
  umount,

  deny @{PROC}/* w,   # deny write for all files directly in /proc (not in a subdir)
  # deny write to files not in /proc/<number>/** or /proc/sys/**
  deny @{PROC}/{[^1-9],[^1-9][^0-9],[^1-9s][^0-9y][^0-9s],[^1-9][^0-9][^0-9][^0-9]*}/** w,
  deny @{PROC}/sys/[^k]** w,  # deny /proc/sys except /proc/sys/k* (effectively /proc/sys/kernel)
  deny @{PROC}/sys/kernel/{?,??,[^s][^h][^m]**} w,  # deny everything except shm* in /proc/sys/kernel/
  deny @{PROC}/sysrq-trigger rwklx,
  deny @{PROC}/mem rwklx,
  deny @{PROC}/kmem rwklx,
  deny @{PROC}/kcore rwklx,

  deny mount,

  deny /sys/[^f]*/** wklx,
  deny /sys/f[^s]*/** wklx,
  deny /sys/fs/[^c]*/** wklx,
  deny /sys/fs/c[^g]*/** wklx,
  deny /sys/fs/cg[^r]*/** wklx,
  deny /sys/firmware/efi/efivars/** rwklx,
  deny /sys/kernel/security/** rwklx,


  # suppress ptrace denials when using 'docker ps' or using 'ps' inside a container
  ptrace (trace,read) peer=docker-default,

  # END docker-default

  # Audit all writes!
  audit /** w,
}
`
)

const (
	parser     = "apparmor_parser"
	journalctl = "journalctl"
	apparmorfs = "/sys/kernel/security/apparmor"
)

var (
	proxyLogs = flag.Bool("proxy-logs", false,
		"Whether to scrape the journal logs for audit events, and print to stdout")
	profileName = flag.String("profile", "docker-default",
		"Name of the profile to install")
)

func main() {
	flag.Parse()

	checkAppArmorDependencies()
	if *proxyLogs {
		checkJournalDependencies()
	}

	if err := loadAuditProfile(); err != nil {
		log.Fatal(err)
	}

	if *proxyLogs {
		doProxyLogs()
	} else {
		// Process runs in DaemonSet, so never exit.
		log.Printf("Sleeping...")
		time.Sleep(365 * 24 * time.Hour)
	}

	log.Fatalf("Unexpected termination")
}

func checkAppArmorDependencies() {
	// Must have permission to access profiles. This is a rough approximation.
	f, err := os.OpenFile(path.Join(apparmorfs, "profiles"), os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("Unable to access AppArmor profiles: %v", err)
	}
	if err := f.Close(); err != nil {
		log.Printf("Error closing probe file: %v", err)
	}

	// Check that the required parser binary is found.
	if _, err := exec.LookPath(parser); err != nil {
		log.Fatalf("Required binary %s not found in PATH", parser)
	}
}

func checkJournalDependencies() {
	// Check for required journalctl binary.
	if _, err := exec.LookPath(journalctl); err != nil {
		log.Fatalf("Required binary %s not found in PATH", journalctl)
	}

	// Check for system journal read permission
	if out, err := exec.Command(journalctl, "--system", "-q").CombinedOutput(); err != nil {
		log.Fatalf("Failed to read system journal: %v", err)
	} else if strings.HasPrefix(string(out), "No journal files were opened due to insufficient permissions.") {
		log.Fatalf("Insufficient permissions to read journal log: %s", out)
	}
}

func loadAuditProfile() error {
	// Write profile to a temporary location.
	data := []byte(strings.Replace(profile, profileNameVar, *profileName, 1))
	if err := ioutil.WriteFile(profilePath, data, 0444); err != nil {
		return fmt.Errorf("Failed to write profile file: %v", err)
	}

	cmd := exec.Command(parser, "--verbose", "--replace", "--skip-cache", profilePath)
	log.Printf("Loading profile: %s", data)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	out, err := cmd.Output()
	fmt.Printf("%s", out)
	if err != nil || stderr.Len() > 0 {
		return fmt.Errorf("Failed to load profile: %v\n%s", err, stderr)
	}

	log.Printf("Profile successfully loaded.")
	return nil
}

func doProxyLogs() {
	log.Printf("Beginning journal proxy --")
	cmd := exec.Command(journalctl, "--directory=/var/log/journal", "--identifier=audit", "--follow")
	cmd.Stdout = os.Stdout // Print output directly.
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Error following journal logs: %v", err)
	}
}
