//Package sdu reports the size of folders
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	// "os"
	"path/filepath"
	"time"
)

var timed = flag.Bool("t", false, "Report execution time")

// File size units
const (
	kB = 1e3
	MB = 1e6
	GB = 1e9
)

func main() {
	// Parse command line flags and arguments
	flag.Parse()
	targetDirs := flag.Args()
	if len(targetDirs) == 0 {
		targetDirs = append(targetDirs, ".")
	}

	// If the -t command flag is set report the total execution time
	if *timed {
		defer timeExec()()
	}

	for _, dir := range targetDirs {
		var totalSize int64
		totalSize += dirSize(dir)
		fmt.Printf("%s: %s\n", dir, formatFileSize(totalSize))
	}
}

func dirSize(dirName string) (totalSize int64) {
	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		fmt.Println("error reading", err)
	}
	for _, file := range files {
		totalSize += file.Size()
		if file.IsDir() {
			totalSize += dirSize(filepath.Join(dirName, file.Name()))
		}
	}
	return
}

// formatFileSize parses a size in bytes into an appropriate unit
func formatFileSize(size int64) string {
	switch {
	case size/GB > 0:
		return fmt.Sprintf("%6.2f GB", float64(size)/GB)
	case size/MB > 0:
		return fmt.Sprintf("%6.2f MB", float64(size)/MB)
	case size/kB > 0:
		return fmt.Sprintf("%6.2f kB", float64(size)/kB)
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}

// timeExec prints the total execution time
func timeExec() func() {
	start := time.Now()
	return func() {
		fmt.Printf("%.2fs elapsed\n", time.Since(start).Seconds())
	}
}
