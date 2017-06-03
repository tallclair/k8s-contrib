/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
)

var (
	dirs     = pflag.StringSliceP("dir", "d", []string{}, "Paths to directories to watch for new AppArmor profiles.")
	interval = pflag.DurationP("interval", "i", 30*time.Second, "Interval for re-listing the watched directories.")
)

const (
	parser     = "apparmor_parser"
	apparmorfs = "/sys/kernel/security/apparmor"
)

func main() {
	pflag.Parse()

	// Check that the required parser binary is found.
	if _, err := exec.LookPath(parser); err != nil {
		glog.Exitf("Required binary %s not found in PATH", parser)
	}

	// Check that loaded profiles can be read.
	if _, err := getLoadedProfiles(); err != nil {
		glog.Exitf("Unable to access apparmor profiles: %v", err)
	}

	ticker := time.NewTicker(*interval)
	for _ = range ticker.C {
		loadNewProfiles()
	}
}

func loadNewProfiles() {
	loadedProfiles, err := getLoadedProfiles()
	if err != nil {
		glog.Errorf("Error reading loaded profiles: %v", err)
		return
	}

	for _, dir := range *dirs {
		infos, err := ioutil.ReadDir(dir)
		if err != nil {
			glog.Warningf("Error reading %s: %v", dir, err)
			continue
		}

		for _, info := range infos {
			if info.IsDir() {
				// Directory listing is shallow.
				continue
			}

			path := filepath.Join(dir, info.Name())
			profiles, err := getProfileNames(path)
			if err != nil {
				glog.Warningf("Error reading %s: %v", path, err)
				continue
			}

			if unloadedProfiles(loadedProfiles, profiles) {
				if err := loadProfiles(path); err != nil {
					glog.Errorf("Could not load profiles: %v", err)
					continue
				}
				// Add new profiles to list of loaded profiles.
				for _, profile := range profiles {
					loadedProfiles[profile] = true
				}
			}
		}
	}
}

func getProfileNames(path string) ([]string, error) {
	cmd := exec.Command(parser, "--names", path)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error reading profiles from %s: %v", path, err)
	}
	return strings.Split(string(out), "\n"), nil
}

func unloadedProfiles(loadedProfiles map[string]bool, profiles []string) bool {
	for _, profile := range profiles {
		if !loadedProfiles[profile] {
			return true
		}
	}
	return false
}

func loadProfiles(path string) error {
	cmd := exec.Command(parser, "--write-cache", path)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error loading profiles from %s: %v", path, err)
	}
	glog.V(2).Infof("Loaded profiles from %s: %v", path, out)
	return nil
}

// FIXME - everything below should be a vendor library from k8s.
func getLoadedProfiles() (map[string]bool, error) {
	profilesPath := path.Join(apparmorfs, "profiles")
	profilesFile, err := os.Open(profilesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %v", profilesPath, err)
	}
	defer profilesFile.Close()

	profiles := map[string]bool{}
	scanner := bufio.NewScanner(profilesFile)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			// Unknown line format; skip it.
			continue
		}
		profiles[fields[0]] = true
	}
	return profiles, nil
}
