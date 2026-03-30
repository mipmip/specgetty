package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type FileEntry struct {
	Path  string
	IsDir bool
}

type ProjectStatus struct {
	Files    []FileEntry
	ScanTime time.Duration
}

type ProjectMap map[string]ProjectStatus

type Config struct {
	ScanDirs struct {
		Include []string `yaml:"include"`
		Exclude []string `yaml:"exclude"`
	} `yaml:"scandirs"`
	FollowSymlinks bool   `yaml:"followsymlinks"`
	EditCommand    string `yaml:"edit_command"`
}

func DumpConfig(config *Config) error {
	b, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func ParseConfigFile(filename, defaultConfig string) (*Config, error) {
	b, err := ioutil.ReadFile(filepath.Clean(filename))
	switch {
	case err == nil:
	case os.IsNotExist(err):
		b = ([]byte)(defaultConfig)
	default:
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// ListOpenSpecContents recursively reads the openspec/ directory under dir
// and returns relative paths with a dir/file indicator.
func ListOpenSpecContents(dir string) ([]FileEntry, error) {
	openspecDir := filepath.Join(dir, "openspec")
	var entries []FileEntry

	err := filepath.WalkDir(openspecDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return errors.Wrap(err, path)
		}
		rel, err := filepath.Rel(openspecDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		entries = append(entries, FileEntry{
			Path:  rel,
			IsDir: d.IsDir(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// Scan finds all OpenSpec projects in directories specified by config
func Scan(config *Config, ignore_dir_errors bool) (ProjectMap, error) {
	ctx := context.Background()
	projects := make(chan string, 1000)

	type walkResult struct {
		err      error
		duration time.Duration
	}
	ch := make(chan walkResult)
	go func() {
		start := time.Now()
		err := Walk(ctx, config, projects, ignore_dir_errors)
		ch <- walkResult{
			err:      err,
			duration: time.Since(start),
		}
	}()

	results := make(ProjectMap)
	totalScanDuration := time.Duration(0)
	for d := range projects {
		start := time.Now()

		files, err := ListOpenSpecContents(d)
		if err != nil {
			return nil, err
		}

		duration := time.Since(start)
		log.Println(d, duration)

		totalScanDuration += duration
		results[d] = ProjectStatus{
			Files:    files,
			ScanTime: duration,
		}
	}

	w := <-ch
	log.Println("walkDuration:", w.duration)
	log.Println("scanDuration:", totalScanDuration)
	return results, w.err
}
