package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

type ScannerDuration struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration"`
}

func main() {
	dirName := "timings"

	// Read all files in the directory
	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		log.Fatal(err)
	}

	// Slice to store scanner durations
	var scannerDurations []ScannerDuration

	for _, file := range files {
		filename := file.Name()

		// Extract scanner name from filename: Assuming format is fixed as shown
		parts := strings.Split(filename, "_")
		if len(parts) < 3 {
			continue // Skip files that do not match the expected format
		}
		scannerName := strings.Join(parts[2:len(parts)-2], "_")

		// Read the content of the file
		content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", dirName, filename))
		if err != nil {
			log.Println("Error reading file:", filename, err)
			continue
		}

		// Parse the duration from the file content
		duration, err := time.ParseDuration(string(content))
		if err != nil {
			log.Println("Error parsing duration from file:", filename, err)
			continue
		}

		// Append the scanner name and duration to the slice
		scannerDurations = append(scannerDurations, ScannerDuration{Name: scannerName, Duration: duration})
	}

	// Convert the slice to a JSON object
	jsonData, err := json.Marshal(scannerDurations)
	if err != nil {
		log.Fatal("Error marshalling JSON:", err)
	}

	fmt.Println(string(jsonData))
}
