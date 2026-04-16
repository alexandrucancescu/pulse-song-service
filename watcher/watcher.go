package watcher

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/unicode"
)

// OnChangeFunc is called when the watched file changes with new content.
type OnChangeFunc func(content string)

// Watch monitors a file for changes, reads and decodes its content,
// and calls onChange when the content actually changes.
// It blocks until the stop channel is closed.
func Watch(filePath string, onChange OnChangeFunc, stop <-chan struct{}) error {
	// Verify the file exists before starting.
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("cannot access watched file: %w", err)
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("cannot create file watcher: %w", err)
	}
	defer w.Close()

	if err := w.Add(filePath); err != nil {
		return fmt.Errorf("cannot watch file %s: %w", filePath, err)
	}

	log.Printf("listening for file changes: %s", filePath)

	var currentContent string

	for {
		select {
		case <-stop:
			return nil

		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			// Only react to writes (file content changed).
			if !event.Has(fsnotify.Write) {
				continue
			}

			content, err := readAndDecode(filePath)
			if err != nil {
				log.Printf("ERROR: reading file: %v", err)
				continue
			}

			content = strings.TrimSpace(content)

			if content == "" || content == currentContent {
				continue
			}

			currentContent = content
			onChange(content)

		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			log.Printf("ERROR: file watcher: %v", err)
		}
	}
}

// readAndDecode reads the file and decodes it from its detected encoding to UTF-8.
func readAndDecode(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	if len(data) == 0 {
		return "", nil
	}

	enc, encName := detectEncoding(data)

	decoder := enc.NewDecoder()
	decoded, err := decoder.Bytes(data)
	if err != nil {
		// If decoding fails, fall back to reading as raw UTF-8.
		log.Printf("WARNING: decoding from %s failed, using raw bytes: %v", encName, err)
		return string(data), nil
	}

	log.Printf("file read (encoding: %s): %s", encName, strings.TrimSpace(string(decoded)))
	return string(decoded), nil
}

// detectEncoding tries to determine the encoding of the data.
// Returns the encoding and its name. Falls back to UTF-8 if detection fails.
func detectEncoding(data []byte) (encoding.Encoding, string) {
	// charset.DetermineEncoding is designed for HTML but works well for
	// general text encoding detection — it examines byte patterns and BOMs.
	_, name, certain := charset.DetermineEncoding(data, "")

	if name != "" && (certain || len(data) > 10) {
		enc, err := htmlindex.Get(name)
		if err == nil && enc != nil {
			log.Printf("detected encoding: %s (certain: %v)", name, certain)
			return enc, name
		}
	}

	log.Printf("encoding detection uncertain, defaulting to UTF-8")
	return unicode.UTF8, "utf-8"
}
