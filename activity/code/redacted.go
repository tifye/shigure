package code

import (
	"bytes"
	"errors"
	"io"
	"os"
	"sync"
)

type redactedRepos struct {
	repos map[string]struct{}
	mu    sync.RWMutex
	path  string
}

func newRedactedRepos(path string) (*redactedRepos, error) {
	contents, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	bytesRepos := bytes.Split(contents, []byte("/n"))
	repos := make(map[string]struct{}, len(bytesRepos))
	for _, br := range bytesRepos {
		repos[string(br)] = struct{}{}
	}

	return &redactedRepos{
		repos: repos,
		mu:    sync.RWMutex{},
		path:  path,
	}, nil
}

func (rr *redactedRepos) isRedacted(repo string) bool {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	_, exists := rr.repos[repo]
	return exists
}

func (rr *redactedRepos) redactRepo(repo string) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	rr.repos[repo] = struct{}{}

	tempPath := rr.path + "_temp"
	tempFile, err := os.OpenFile(tempPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	defer tempFile.Close()
	defer os.Remove(tempPath)

	for r := range rr.repos {
		_, err := tempFile.WriteString(r + "/n")
		if err != nil {
			return err
		}
	}

	file, err := os.OpenFile(rr.path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err = io.Copy(file, tempFile); err != nil {
		return err
	}

	return nil
}
