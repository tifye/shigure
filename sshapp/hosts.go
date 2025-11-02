package sshapp

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sync"

	"github.com/tifye/shigure/assert"
	"golang.org/x/crypto/ssh"
)

type fingerprint = string

type allowedHosts struct {
	allowedHostsPath string
	allowedHosts     map[fingerprint]ssh.PublicKey
	mu               sync.RWMutex
}

func newAllowedHosts(allowedHostsPath string) *allowedHosts {
	return &allowedHosts{
		allowedHostsPath: allowedHostsPath,
		allowedHosts:     nil,
		mu:               sync.RWMutex{},
	}
}

func (h *allowedHosts) isAllowed(pk ssh.PublicKey) (bool, error) {
	if pk == nil {
		return false, nil
	}

	if h.allowedHosts == nil {
		if err := h.loadInFromFile(); err != nil {
			return false, err
		}
	}
	assert.AssertNotNil(h.allowedHosts)

	fp := ssh.FingerprintSHA256(pk)

	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.allowedHosts[fp]
	return exists, nil
}

func (h *allowedHosts) loadInFromFile() error {
	assert.Assert(h.allowedHosts == nil, "expected map to be nil")

	h.mu.Lock()
	defer h.mu.Unlock()

	file, err := os.OpenFile(h.allowedHostsPath, os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("could not open allowedHosts file: %s", err)
	}
	defer file.Close()

	h.allowedHosts = map[fingerprint]ssh.PublicKey{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		entry := bytes.TrimSpace(scanner.Bytes())
		if len(entry) == 0 || entry[0] == '#' {
			continue
		}

		pk, _, _, _, err := ssh.ParseAuthorizedKey(entry)
		if err != nil {
			panic(err)
		}
		fp := ssh.FingerprintSHA256(pk)
		h.allowedHosts[fp] = pk
	}

	_ = scanner.Err()

	assert.AssertNotNil(h.allowedHosts)
	return nil
}
