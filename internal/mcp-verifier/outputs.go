package mcpverifier

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

func createTempFile(experimentType, experiment string) (*os.File, error) {
	if _, err := os.Stat(tmpFileDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(tmpFileDir, 0700); err != nil {
				return nil, err
			}
		}
	}
	file, err := os.CreateTemp(tmpFileDir, fmt.Sprintf("%s-%s-*.json", experimentType, experiment))
	if err != nil {
		return nil, err
	}
	return file, nil
}

func mergeTempJSONFilesStreaming(dir, experimentType, experiment string) (string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	fileCh := make(chan string, 100)
	dataCh := make(chan map[string][]ToolResponses, 100)

	workerCount := runtime.NumCPU()
	var wg sync.WaitGroup

	// Worker pool
	for range workerCount {
		wg.Add(1)

		go func() {
			defer wg.Done()

			// Will wait till there is a file saved in the channel
			// feed by the other concurrent read on `range files`
			for path := range fileCh {
				data, err := os.ReadFile(path)
				if err != nil {
					continue // skip corrupted files
				}

				var tmp map[string][]ToolResponses
				if err := json.Unmarshal(data, &tmp); err != nil {
					continue
				}

				dataCh <- tmp

				// cleanup temp file
				_ = os.Remove(path)
			}
		}()
	}

	// Feed files to workers
	go func() {
		prefix := fmt.Sprintf("%s-%s", experimentType, experiment)

		for _, f := range files {
			if f.IsDir() {
				continue
			}

			if !strings.HasPrefix(f.Name(), prefix) {
				continue
			}

			fileCh <- filepath.Join(dir, f.Name())
		}

		close(fileCh)
	}()

	// Close data channel after workers finish
	go func() {
		wg.Wait()
		close(dataCh)
	}()

	// Now we merge the files from the data channel
	merged := make(map[string][]ToolResponses)

	for m := range dataCh {
		for k, v := range m {
			merged[k] = append(merged[k], v...)
		}
	}

	timestamp := time.Now().UTC().Format("20060102-150405.000")
	outPath := filepath.Join(dir, fmt.Sprintf("%s-%s-%s.json", experimentType, experiment, timestamp))

	outJSON, err := json.Marshal(merged)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(outPath, outJSON, 0644)
	if err != nil {
		return "", err
	}

	return outPath, nil
}
