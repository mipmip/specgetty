package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	taskDoneRe   = regexp.MustCompile(`(?m)^- \[x\] `)
	taskUndoneRe = regexp.MustCompile(`(?m)^- \[ \] `)
)

// ParseTaskStats counts done and total checkbox tasks in markdown content.
func ParseTaskStats(content string) (total, done int) {
	done = len(taskDoneRe.FindAllString(content, -1))
	undone := len(taskUndoneRe.FindAllString(content, -1))
	total = done + undone
	return
}

type FileEntry struct {
	Path  string
	IsDir bool
}

type ChangeInfo struct {
	Name             string
	ArtifactFiles    []string          // sorted .md filenames
	ArtifactContents map[string]string // filename → content
	TasksTotal       int
	TasksDone        int
	SpecNames        []string
	SpecContents     map[string]string
	ArchiveDate      time.Time // zero for active changes, mtime for archived
}

type ProjectInfo struct {
	SpecCount       int
	SpecNames       []string
	SpecContents    map[string]string
	ActiveChanges   []string
	Changes         []ChangeInfo
	ArchivedChanges []ChangeInfo
	ConfigFile      string // "project.md", "config.yaml", or ""
	ConfigContent   string
	TasksTotal      int // aggregate across all active changes
	TasksDone       int
}

type ProjectStatus struct {
	Files    []FileEntry
	Info     ProjectInfo
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

// parseChangeDir reads a change directory and returns a ChangeInfo.
func parseChangeDir(dir string, name string) ChangeInfo {
	ci := ChangeInfo{
		Name:             name,
		ArtifactContents: make(map[string]string),
		SpecContents:     make(map[string]string),
	}

	// Discover .md files
	if files, err := os.ReadDir(dir); err == nil {
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
				continue
			}
			ci.ArtifactFiles = append(ci.ArtifactFiles, f.Name())
			if b, err := os.ReadFile(filepath.Join(dir, f.Name())); err == nil {
				ci.ArtifactContents[f.Name()] = string(b)
			}
		}
	}
	sort.Strings(ci.ArtifactFiles)

	// Parse task stats from tasks.md if present
	if tasksContent, ok := ci.ArtifactContents["tasks.md"]; ok {
		ci.TasksTotal, ci.TasksDone = ParseTaskStats(tasksContent)
	}

	// Read specs within the change
	specsDir := filepath.Join(dir, "specs")
	if specEntries, err := os.ReadDir(specsDir); err == nil {
		for _, se := range specEntries {
			if se.IsDir() {
				ci.SpecNames = append(ci.SpecNames, se.Name())
				specFile := filepath.Join(specsDir, se.Name(), "spec.md")
				if b, err := os.ReadFile(specFile); err == nil {
					ci.SpecContents[se.Name()] = string(b)
				}
			}
		}
	}
	sort.Strings(ci.SpecNames)

	return ci
}

// ParseProjectInfo reads the openspec/ directory structure to extract stats.
func ParseProjectInfo(dir string) ProjectInfo {
	info := ProjectInfo{}
	openspecDir := filepath.Join(dir, "openspec")

	// Specs: subdirectories in openspec/specs/, read spec.md contents
	specsDir := filepath.Join(openspecDir, "specs")
	info.SpecContents = make(map[string]string)
	if entries, err := os.ReadDir(specsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				info.SpecCount++
				info.SpecNames = append(info.SpecNames, e.Name())
				specFile := filepath.Join(specsDir, e.Name(), "spec.md")
				if b, err := os.ReadFile(specFile); err == nil {
					info.SpecContents[e.Name()] = string(b)
				}
			}
		}
	}
	sort.Strings(info.SpecNames)

	// Active changes: subdirectories in openspec/changes/
	changesDir := filepath.Join(openspecDir, "changes")
	if entries, err := os.ReadDir(changesDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() || e.Name() == "archive" || e.Name() == "discarded" {
				continue
			}
			info.ActiveChanges = append(info.ActiveChanges, e.Name())
			ci := parseChangeDir(filepath.Join(changesDir, e.Name()), e.Name())
			info.Changes = append(info.Changes, ci)
		}
	}
	sort.Slice(info.Changes, func(i, j int) bool {
		return info.Changes[i].Name < info.Changes[j].Name
	})

	// Aggregate task stats across active changes
	for _, ci := range info.Changes {
		info.TasksTotal += ci.TasksTotal
		info.TasksDone += ci.TasksDone
	}

	// Archived changes: subdirectories in openspec/archive/
	archiveDir := filepath.Join(openspecDir, "changes", "archive")
	if entries, err := os.ReadDir(archiveDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				dirName := e.Name()
				displayName := dirName
				var archiveDate time.Time
				// Parse YYYY-MM-DD- prefix from directory name
				if len(dirName) >= 11 && dirName[4] == '-' && dirName[7] == '-' && dirName[10] == '-' {
					if t, err := time.Parse("2006-01-02", dirName[:10]); err == nil {
						archiveDate = t
						displayName = dirName[11:]
					}
				}
				ci := parseChangeDir(filepath.Join(archiveDir, dirName), displayName)
				ci.ArchiveDate = archiveDate
				info.ArchivedChanges = append(info.ArchivedChanges, ci)
			}
		}
	}

	// Config file: project.md takes priority over config.yaml
	projectMd := filepath.Join(openspecDir, "project.md")
	configYaml := filepath.Join(openspecDir, "config.yaml")
	if b, err := os.ReadFile(projectMd); err == nil {
		info.ConfigFile = "project.md"
		info.ConfigContent = string(b)
	} else if b, err := os.ReadFile(configYaml); err == nil {
		info.ConfigFile = "config.yaml"
		info.ConfigContent = string(b)
	}

	return info
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

		info := ParseProjectInfo(d)

		totalScanDuration += duration
		results[d] = ProjectStatus{
			Files:    files,
			Info:     info,
			ScanTime: duration,
		}
	}

	w := <-ch
	log.Println("walkDuration:", w.duration)
	log.Println("scanDuration:", totalScanDuration)
	return results, w.err
}
