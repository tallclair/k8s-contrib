package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	testDir  = flag.String("dir", "/empty", "Target directory to test")
	testFile = flag.String("file", "foo", "Target file to test")
	hang     = flag.Bool("hang", true, "Don't exit")
)

func main() {
	flag.Parse()

	// whoami
	log.Printf("UID: %d", os.Getuid())
	log.Printf("GID: %d", os.Getgid())
	gids, err := os.Getgroups()
	noerr(err)
	log.Printf("Groups: %v", gids)

	// ls -l /$dir
	stat, err := os.Stat(*testDir)
	noerr(err)
	log.Printf("Dir: %13s   %04o", stat.Name(), stat.Mode()&os.ModePerm)

	// touch /$dir/$file
	path := filepath.Join(*testDir, *testFile)
	err = ioutil.WriteFile(path, []byte("foo bar baz\n"), os.ModePerm)
	noerr(err)

	// ls -l /$dir/$file
	stat, err = os.Stat(path)
	noerr(err)
	log.Printf("File: %12s   %04o", stat.Name(), stat.Mode()&os.ModePerm)

	// cat /$dir/$file
	data, err := ioutil.ReadFile(path)
	noerr(err)
	log.Printf("Contents: %q", data)

	if *hang {
		time.Sleep(1 << 60)
	}
}

func noerr(err error) {
	if err != nil {
		log.Panicf("Unexpected error: %v", err)
	}
}
