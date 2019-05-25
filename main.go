package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"text/template"

	log "github.com/sirupsen/logrus"

	"github.com/Luzifer/go_helpers/v2/str"
	"github.com/Luzifer/rconfig/v2"
)

var (
	cfg = struct {
		ConcurrencyLimit int      `flag:"concurrency-limit" default:"50" description:"How many queries to execute in parallel"`
		IANATldList      string   `flag:"iana-tld-list" vardefault:"iana-tld-list" description:"IANA TLD list file"`
		IANAFilter       []string `flag:"iana-filter" vardefault:"iana-filter" description:"IANA TLDs to igore"`
		InternicRootFile string   `flag:"internic-root-file" vardefault:"internic-root" description:"Internic root nameserver file"`
		LogLevel         string   `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		OpenNICFilter    []string `flag:"opennic-filter" vardefault:"opennic-filter" description:"OpenNIC TLDs to ignore"`
		OpenNICRoot      string   `flag:"opennic-root" vardefault:"opennic-root" description:"OpenNIC root server"`
		VersionAndExit   bool     `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	version = "dev"
)

const stubTpl = `# Autogenerated with rootzone {{.version}}
#{{ range $tld, $ips := .roots }}
zone "{{ $tld }}" in {
  type static-stub;
  server-addresses { {{ range $ips }}{{ . }}; {{ end }}};
};
#{{ end }}

# vim: set ft=named:
`

func init() {
	rconfig.SetVariableDefaults(map[string]string{
		"iana-filter":    strings.Join([]string{"arpa."}, ","),
		"iana-tld-list":  "https://data.iana.org/TLD/tlds-alpha-by-domain.txt",
		"internic-root":  "https://www.internic.net/domain/named.root",
		"opennic-filter": strings.Join([]string{".", "opennic.glue."}, ","),
		"opennic-root":   "75.127.96.89:53",
	})
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		log.Fatalf("Unable to parse commandline options: %s", err)
	}

	if cfg.VersionAndExit {
		fmt.Printf("rootzone %s\n", version)
		os.Exit(0)
	}

	if l, err := log.ParseLevel(cfg.LogLevel); err != nil {
		log.WithError(err).Fatal("Unable to parse log level")
	} else {
		log.SetLevel(l)
	}
}

func main() {
	var (
		cLimiter         = make(chan struct{}, cfg.ConcurrencyLimit)
		rootServersMutex = new(sync.Mutex)
		wg               = new(sync.WaitGroup)
	)

	// Initialize nameserver list before first request
	getRandomInternicRoot()

	rootServers := map[string][]string{
		"opennic.glue.": {cfg.OpenNICRoot},
	}

	// Fetch IANA TLDs
	ianaTLDs, err := getIANATLDs()
	if err != nil {
		log.WithError(err).Fatal("Unable to retrieve IANA TLDs")
	}
	setRootsFromTLDs(rootServers, rootServersMutex, ianaTLDs, cfg.IANAFilter, getIANAZoneMasters, wg, cLimiter)

	// Fetch OpenNIC TLDs
	opennicTLDs, err := getOpenNICTLDs()
	if err != nil {
		log.WithError(err).Fatal("Unable to retrieve OpenNIC TLDs")
	}
	setRootsFromTLDs(rootServers, rootServersMutex, opennicTLDs, cfg.OpenNICFilter, getOpenNICZoneMasters, wg, cLimiter)

	wg.Wait()

	tpl, err := template.New("stub").Parse(stubTpl)
	if err != nil {
		log.WithError(err).Fatal("Unable to parse template")
	}

	if err := tpl.Execute(os.Stdout, map[string]interface{}{
		"roots":   rootServers,
		"version": version,
	}); err != nil {
		log.WithError(err).Fatal("Unable to generate stub file")
	}
}

func setRootsFromTLDs(roots map[string][]string, rootsMutex *sync.Mutex, tlds []string, filter []string, resolver func(string) ([]string, error), wg *sync.WaitGroup, cLimiter chan struct{}) {
	for _, tld := range tlds {
		if !strings.HasSuffix(tld, ".") {
			tld = tld + "."
		}

		if str.StringInSlice(tld, filter) {
			continue
		}

		wg.Add(1)
		cLimiter <- struct{}{}

		go func(tld string) {
			defer func() {
				wg.Done()
				<-cLimiter
			}()

			masters, err := resolver(tld)
			if err != nil {
				log.WithError(err).WithField("zone", tld).Error("Unable to retrieve zone masters")
				return
			}

			rootsMutex.Lock()
			defer rootsMutex.Unlock()

			sort.Strings(masters)

			roots[tld] = masters
		}(tld)
	}
}
