package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Luzifer/go_helpers/rand"
	"github.com/sirupsen/logrus"
)

const internicRootTimeout = 2 * time.Second

var (
	internicRoots      []string
	internicRootsFetch sync.Mutex
)

func getRandomInternicRoot(ctx context.Context) (root string, err error) {
	internicRootsFetch.Lock()
	defer internicRootsFetch.Unlock()

	if len(internicRoots) == 0 {
		// Initialize InterNIC root cache
		reqCtx, cancel := context.WithTimeout(ctx, internicRootTimeout)
		defer cancel()

		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, cfg.InternicRootFile, nil)
		if err != nil {
			return "", fmt.Errorf("creating InterNIC root request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("requesting InterNIC roots: %w", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				logrus.WithError(err).Error("closing InterNIC request")
			}
		}()

		var (
			matcher = regexp.MustCompile(`\s+A\s+([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)$`)
			roots   []string
			scanner = bufio.NewScanner(resp.Body)
		)

		for scanner.Scan() {
			m := matcher.FindStringSubmatch(scanner.Text())
			if len(m) != 2 {
				continue
			}
			roots = append(roots, strings.Join([]string{m[1], "53"}, ":"))
		}

		if err = scanner.Err(); err != nil {
			return "", fmt.Errorf("scanning root list: %w", err)
		}

		internicRoots = roots
	}

	root, err = rand.EntryFromSlice(internicRoots)
	if err != nil {
		return "", fmt.Errorf("selecting random root-server: %w", err)
	}

	return root, nil
}
