//Package sdu reports the size of folders
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// File size units
const (
	kB = 1e3
	MB = 1e6
	GB = 1e9
)

// Zero size struct for signal channels
type nop struct{}

// Limit number of open files
var n = make(chan nop, 30)

// Cancellation channel
var abort = make(chan nop)

// Timer channel
var timer <-chan time.Time

// Flags
var timed = flag.Bool("t", false, "Report execution time")
var verbose = flag.Bool("v", false, "Print intermediary result")

func main() {
	// Multiple folders can be passed as the arguments.
	// Default argument is the current folder
	flag.Parse()
	targetDirs := flag.Args()
	if len(targetDirs) == 0 {
		targetDirs = append(targetDirs, ".")
	}

	// Sum up sizes sent over the channel from traversing goroutines
	var totalSize int64

	// If the -t command flag is set report the total execution time
	if *timed {
		defer timeExec()()
	}
	// If -v command flag is set, start timer
	if *verbose {
		timer = time.Tick(time.Second)
		go func() {
			for {
				select {
				case <-timer:
					fmt.Printf("Total size: %s\n", formatFileSize(totalSize))
				case <-abort:
					return
				}
			}
		}()
	}

	// Concurrent directory traversal sends results through the sizes channel
	sizes := make(chan int64)
	// The WaitGroup counts traversing sub-goroutines
	var wg sync.WaitGroup

	// Launch goroutine to traverse each directory
	for _, dir := range targetDirs {
		wg.Add(1)
		go dirSize(dir, &wg, sizes)
	}

	// When all directory sizes have been computed we can finish accumulating
	go func() {
		wg.Wait()
		close(sizes)
	}()

	// Check if user has canceled
	go pollAbort()

loop:
	for {
		select {
		case s, ok := <-sizes:
			if !ok {
				break loop
			}
			totalSize += s
		case <-abort:
			fmt.Println("Canceled.")
			fmt.Printf("Total size up to now: %s\n", formatFileSize(totalSize))
			// Make sure all goroutines have finished and sizes is closed
			for range sizes {
			}
			return
		}
	}
	fmt.Printf("Total size: %s\n", formatFileSize(totalSize))
}

// dirSize calculates the size of a directory recursively
func dirSize(dirName string, wg *sync.WaitGroup, sizes chan<- int64) {
	defer wg.Done()
	if cancelled() {
		return
	}
	for _, file := range readDir(dirName) {
		if cancelled() {
			return
		}
		sizes <- file.Size()
		if file.IsDir() {
			wg.Add(1)
			go dirSize(filepath.Join(dirName, file.Name()), wg, sizes)
		}
	}
}

// pollAbort waits for user input to cancel execution
func pollAbort() {
	os.Stdin.Read(make([]byte, 1))
	close(abort)
}

// readDir returns file infos for a given dictionary
// Writing to the n channel guarantees that not more than
// cap(n) directories are read concurrently
func readDir(dir string) []os.FileInfo {
	select {
	case n <- nop{}:
	case <-abort:
		return nil
	}
	defer func() { <-n }()
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		return nil
	}
	return files
}

// Helper to check if user has cancelled
func cancelled() bool {
	select {
	case <-abort:
		return true
	default:
		return false
	}
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
