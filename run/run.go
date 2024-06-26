package run

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	kcl "kcl-lang.io/kcl-go"
)

func Run(manifestPath, outdir string) {
	if isURL(manifestPath) {
		var err error
		manifestPath, err = resolveManifestPath(manifestPath)
		if err != nil {
			fmt.Printf("Error resolving manifest path: %v\n", err)
			return
		}
	}

	result, err := kcl.Run(manifestPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	r, err := result.First().ToMap()
	if err != nil {
		fmt.Println(err)
		return
	}
	examplesInterface, ok := r["examples"]
	if !ok {
		fmt.Println("Error: 'examples' field not found in KCL result")
		return
	}
	examples, ok := examplesInterface.([]interface{})
	if !ok {
		fmt.Println("Error: 'examples' field is not a slice of interfaces")
		return
	}
	for i, example := range examples {
		exampleMap, ok := example.(map[string]interface{})
		if !ok {
			fmt.Println("Error: example is not a map")
			continue
		}

		nameInterface, ok := exampleMap["name"]
		if !ok {
			fmt.Println("Error: 'name' field not found in example")
			continue
		}
		name, ok := nameInterface.(string)
		if !ok {
			fmt.Println("Error: 'name' field is not a string")
		}

		txtarInterface, ok := exampleMap["txtar"]
		if !ok {
			fmt.Println("Error: 'txtar' field not found in example")
			continue
		}
		txtar, ok := txtarInterface.(string)
		if !ok {
			fmt.Println("Error: 'txtar' field is not a string")
			continue
		}

		var notes string
		notesInterface, ok := exampleMap["notes"]
		if ok {
			notes, ok = notesInterface.(string)
			if !ok {
				fmt.Println("Error: 'notes' field is not a string")
				continue
			}
		}

		ordinal := i + 1
		dirName := filepath.Join(outdir, fmt.Sprintf("example-%03d", ordinal))
		err := os.MkdirAll(dirName, os.ModePerm)
		if err != nil {
			fmt.Printf("Error creating directory %s: %v\n", dirName, err)
			continue
		}
		manifestFilePath := filepath.Join(dirName, "manifest.txtar")
		manifestFile, err := os.Create(manifestFilePath)
		if err != nil {
			fmt.Printf("Error creating file %s: %v\n", manifestFilePath, err)
			continue
		}
		defer manifestFile.Close()
		_, err = manifestFile.WriteString(strings.TrimSpace(txtar))
		if err != nil {
			fmt.Printf("Error writing to file %s: %v\n", manifestFilePath, err)
			continue
		}
		notesFilePath := filepath.Join(dirName, "notes.txt")
		notesFile, err := os.Create(notesFilePath)
		if err != nil {
			fmt.Printf("Error creating file %s: %v\n", notesFilePath, err)
			continue
		}
		defer notesFile.Close()
		_, err = notesFile.WriteString(strings.TrimSpace(notes))
		if err != nil {
			fmt.Printf("Error writing to file %s: %v\n", notesFilePath, err)
			continue
		}

		nameFilePath := filepath.Join(dirName, "name.txt")
		nameFile, err := os.Create(nameFilePath)
		if err != nil {
			fmt.Printf("Error creating file %s: %v\n", nameFilePath, err)
			continue
		}
		defer nameFile.Close()
		_, err = nameFile.WriteString(strings.TrimSpace(name) + "\n")
		if err != nil {
			panic(err)
		}

		if notes != "" {
			notesFilePath := filepath.Join(dirName, "notes.txt")
			notesFile, err := os.Create(notesFilePath)
			if err != nil {
				fmt.Printf("Error creating file %s: %v\n", notesFilePath, err)
				continue
			}
			defer notesFile.Close()
			_, err = notesFile.WriteString(strings.TrimSpace(notes))
			if err != nil {
				panic(err)
			}
		}

		fmt.Printf("Processed example %d\n", ordinal)
	}
}

func isURL(path string) bool {
	u, err := url.Parse(path)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https" || u.Scheme == "file")
}

func resolveManifestPath(path string) (string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http", "https":
		tempFile, err := downloadManifest(path)
		if err != nil {
			return "", err
		}
		defer tempFile.Close()
		return tempFile.Name(), nil
	case "file":
		return u.Path, nil
	default:
		return "", fmt.Errorf("unsupported protocol scheme: %s", u.Scheme)
	}
}

func downloadManifest(url string) (*os.File, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	tempFile, err := os.CreateTemp("", "manifest-*.k")
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, err
	}
	err = tempFile.Sync()
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, err
	}
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, err
	}
	return tempFile, nil
}
