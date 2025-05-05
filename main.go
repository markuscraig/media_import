package main

// Media Importer CLI Tool
// ---------------------------------------------
// Recursively imports supported photo and video files from a given
// input directory, and copies them to an output directory based on
// a user-defined template (e.g. by year, month, day, and filename).
// Includes support for parallel workers, dry-run mode, customizable
// file extensions, chunk size control, and terminal progress bars.

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"sync"
	"text/template"
	"time"

	"slices"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// cli argument variables
var (
	inputDir     string // Input directory root
	outputDir    string // Output directory root
	templateStr  string // Template for organizing output paths
	extensions   string // Comma-separated list of file extensions to import
	workers      int    // Number of parallel workers
	chunkSizeStr string // Copy buffer size (e.g. 32k, 1m)
	dryRun       bool   // Log actions without copying files
)

// Default supported media file extensions which can
// be overridden by user cli option.
var defaultExtensions = []string{
	".jpg", ".jpeg", ".png", ".gif", ".bmp",
	".mp4", ".mov", ".avi", ".mkv",
}

// FileJob holds the data needed to copy a file
// from source to destination.
type FileJob struct {
	SrcPath string // Full path to the source file
	DstPath string // Full path to the destination file
	Size    int64  // File size in bytes
}

// main is the application entry point. It parses flags, launches
// workers, tracks file copying, and reports performance stats.
func main() {
	// parse command line flags
	flag.StringVar(&inputDir, "input", "", "Input directory root (required)")
	flag.StringVar(&outputDir, "output", "", "Output directory root (required)")
	flag.StringVar(&templateStr, "template", "{{.Year}}/{{.Year}}-{{.Month}}-{{.Day}}/{{.Name}}", "Text template for organizing output paths")
	flag.StringVar(&extensions, "extensions", strings.Join(defaultExtensions, ","), "Comma-separated list of file extensions to import")
	flag.IntVar(&workers, "workers", 3, "Number of parallel workers")
	flag.BoolVar(&dryRun, "dry-run", false, "Log actions without copying files")
	flag.StringVar(&chunkSizeStr, "chunk-size", "1m", "Copy buffer size (e.g. 32k, 1m)")
	flag.Parse()

	// validate input and output base directories
	if inputDir == "" || outputDir == "" {
		log.Fatal("Both input and output directories must be specified")
	}

	// parse the input read chunk size
	chunkSize, err := parseChunkSize(chunkSizeStr)
	if err != nil {
		log.Fatalf("Invalid chunk size: %v", err)
	}

	// parse the file extensions into a slice
	extList := strings.Split(extensions, ",")
	for i := range extList {
		extList[i] = strings.ToLower(strings.TrimSpace(extList[i]))
	}

	// parse the output template directory string
	tpl, err := template.New("pathTemplate").Parse(templateStr)
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	// init the file jobs channel and worker pool
	jobs := make(chan FileJob, 100)
	ctx := context.Background()
	var wg sync.WaitGroup

	// create the text progress bars (only if not dry-run)
	var progress *mpb.Progress
	if !dryRun {
		progress = mpb.New(mpb.WithWaitGroup(&wg))
	}

	// create the aggregate stats shared across all workers
	start := time.Now()
	stats := NewStats()

	// create the worker pool where each worker processes file-copy
	// jobs from the jobs channel.
	for i := range workers {
		wg.Add(1)
		go worker(ctx, i, jobs, dryRun, &wg, stats, progress, chunkSize)
	}

	// traverse the input directory and send file jobs to the workers
	// using the jobs channel.
	err = walkFiles(inputDir, extList, jobs, tpl, outputDir)
	if err != nil {
		log.Fatalf("Error walking input directory: %v", err)
	}

	// wait for all workers to finish processing jobs
	// and close the progress bar if it was created.
	wg.Wait()
	if progress != nil {
		progress.Wait()
	}

	// log the total bytes copied, total time taken, and copy speed
	duration := time.Since(start).Seconds()
	log.Printf("Total copied: %.2f MB in %.2f seconds (%.2f MB/s)",
		stats.MBytes, duration, stats.MBytes/duration)
}

// parseChunkSize parses a human-friendly chunk size string
// (e.g. "32k", "1m") into a byte count.
func parseChunkSize(sizeStr string) (int, error) {
	// remove any whitespace and convert to lowercase
	sizeStr = strings.ToLower(strings.TrimSpace(sizeStr))

	// default to bytes if no suffix is found
	multiplier := 1

	// check for valid suffixes and set the multiplier
	if strings.HasSuffix(sizeStr, "k") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "k")
	} else if strings.HasSuffix(sizeStr, "m") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "m")
	}

	// parse the numeric part of the string
	val, err := strconv.Atoi(sizeStr)
	if err != nil {
		return 0, err
	}

	// return the size in bytes
	return val * multiplier, nil
}

// isSupported returns true if the file has an allowed extension
func isSupported(filePath string, exts []string) bool {
	// check if the file extension is in the list of supported extensions
	ext := strings.ToLower(filepath.Ext(filePath))
	return slices.Contains(exts, ext)
}

// getTemplateData builds a map of metadata values for the template.
func getTemplateData(filePath string, fi os.FileInfo) map[string]any {
	// extract the file modification time
	t := fi.ModTime()

	// create a map with the template data
	return map[string]any{
		"Year":  t.Year(),
		"Month": fmt.Sprintf("%02d", t.Month()),
		"Day":   fmt.Sprintf("%02d", t.Day()),
		"Name":  filepath.Base(filePath),
	}
}

// walkFiles finds files in baseDir and sends jobs to the jobs channel.
func walkFiles(baseDir string, exts []string, jobs chan<- FileJob, tpl *template.Template, outputBase string) error {
	// ensure the output base directory exists
	defer close(jobs)
	absOutputBase, err := filepath.Abs(outputBase)
	if err != nil {
		return err
	}

	// Walk the input directory and send file jobs to the channel
	return filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		// check for errors and skip directories or unsupported files
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				// log permission errors and skip the file
				log.Printf("Permission error: %v", err)
				return nil
			}
			if info.IsDir() || !isSupported(path, exts) {
				return err
			}
		}

		// execute the template with the file metadata
		var buf strings.Builder
		tpl.Execute(&buf, getTemplateData(path, info))

		// create the destination path using the template output
		absDst := filepath.Join(absOutputBase, buf.String())

		// create the file job and send it to the jobs channel
		jobs <- FileJob{SrcPath: path, DstPath: absDst, Size: info.Size()}
		return nil
	})
}

// worker processes file jobs and performs copy with progress bar updates.
// worker processes a stream of file copy jobs. It performs actual file
// copying unless in dry-run mode. It also tracks progress via a progress
// bar and safely updates the shared totalBytes count.
func worker(ctx context.Context, id int, jobs <-chan FileJob, dryRun bool,
	wg *sync.WaitGroup, stats *Stats, progress *mpb.Progress, chunkSize int) {

	// mark the current worker as done when it finishes processing jobs
	defer wg.Done()

	// wait for the next job to be available and process it until the channel
	// is closed or the context is cancelled
	for job := range jobs {
		// check if the source and destination paths are relative
		relSrc := job.SrcPath
		relDst := job.DstPath
		if strings.HasPrefix(job.SrcPath, ".") || strings.HasPrefix(job.DstPath, ".") {
			// get the current working directory and make the paths relative
			if wd, err := os.Getwd(); err == nil {
				if rel, err := filepath.Rel(wd, job.SrcPath); err == nil {
					relSrc = rel
				}
				if rel, err := filepath.Rel(wd, job.DstPath); err == nil {
					relDst = rel
				}
			}
		}

		// if dry-run mode, just log what would be done without copying
		if dryRun {
			log.Printf("[Dry-Run] Worker %d would copy: %s -> %s", id, relSrc, relDst)
			continue
		}

		// create a new progress bar for the current file job
		bar := progress.AddBar(job.Size,
			mpb.PrependDecorators(
				decor.Name(fmt.Sprintf("Worker %d:", id), decor.WC{W: len("Worker 00:"), C: decor.DindentRight}),
				decor.Name(filepath.Base(job.SrcPath)),
			),
			mpb.AppendDecorators(decor.Percentage()),
		)

		// ensure the destination directory exists before writing the file
		if err := os.MkdirAll(filepath.Dir(job.DstPath), 0755); err != nil {
			log.Printf("Error creating directory: %v", err)
			bar.Abort(true)
			continue
		}

		// open the source file for reading
		src, err := os.Open(job.SrcPath)
		if err != nil {
			log.Printf("Error opening source: %v", err)
			bar.Abort(true)
			continue
		}

		// create the destination file for writing
		dst, err := os.Create(job.DstPath)
		if err != nil {
			src.Close()
			log.Printf("Error creating destination: %v", err)
			bar.Abort(true)
			continue
		}

		// copy the file in chunks, updating the progress bar after each write
		buffer := make([]byte, chunkSize)
		var copied int64
		for {
			// read from the source file into the buffer
			nr, er := src.Read(buffer)
			if nr > 0 {
				// write the buffer to the destination file
				nw, ew := dst.Write(buffer[:nr])
				if ew != nil {
					err = ew
					break
				}

				// update the progress bar with the number of bytes written
				copied += int64(nw)
				bar.IncrBy(nw)

				// check if the number of bytes written matches the number read
				if nr != nw {
					err = io.ErrShortWrite
					break
				}
			}

			// check for EOF or other read errors
			if er != nil {
				if er != io.EOF {
					err = er
				}
				break
			}
		}

		// close the source and destination files
		src.Close()
		dst.Close()

		// check for errors during the copy process
		if err != nil {
			// log the error and abort the progress bar
			log.Printf("Error copying file: %v", err)
			bar.Abort(true)
			continue
		}

		// finalize the progress bar after the copy is complete
		bar.SetTotal(copied, true)

		// safely update the total bytes counter
		stats.AddBytes(copied)
	}
}
