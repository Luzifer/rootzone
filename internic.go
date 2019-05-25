package main

import (
	"bufio"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	internicRoots      []string
	internicRootsFetch sync.Mutex
)

func getRandomInternicRoot() string {
	internicRootsFetch.Lock()

	rand.Seed(time.Now().UnixNano())
	if internicRoots != nil {
		internicRootsFetch.Unlock()
		return internicRoots[rand.Intn(len(internicRoots))]
	}

	// Initialize InterNIC root cache
	resp, err := http.Get(cfg.InternicRootFile)
	if err != nil {
		log.WithError(err).Fatal("Unable to get InterNIC root file")
	}
	defer resp.Body.Close()

	var (
		matcher = regexp.MustCompile(`\s+A\s+([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)$`)
		roots   = []string{}
		scanner = bufio.NewScanner(resp.Body)
	)

	for scanner.Scan() {
		m := matcher.FindStringSubmatch(scanner.Text())
		if len(m) != 2 {
			continue
		}
		roots = append(roots, strings.Join([]string{m[1], "53"}, ":"))
	}

	internicRoots = roots

	// Call self which triggers early-exit
	internicRootsFetch.Unlock()
	return getRandomInternicRoot()
}
