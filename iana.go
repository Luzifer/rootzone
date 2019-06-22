package main

import (
	"bufio"
	"net"
	"net/http"
	"strings"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func getIANATLDs() ([]string, error) {
	tlds := []string{}

	resp, err := http.Get(cfg.IANATldList)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve IANA TLD list")
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if b := scanner.Bytes(); b[0] == '#' || len(b) == 0 {
			continue
		}

		tlds = append(tlds, strings.ToLower(scanner.Text())+".")
	}

	return tlds, nil
}

func getIANAZoneMasters(tld string) ([]string, error) {
	log.WithField("tld", tld).Trace("Getting zone masters")
	masters := []string{}

	c := new(dns.Client)
	c.Net = "tcp"

	m := new(dns.Msg)
	m.SetQuestion(tld, dns.TypeNS)

	var (
		err error
		r   *dns.Msg
	)

	if err = retry(func() error {
		r, _, err = c.Exchange(m, getRandomInternicRoot())
		return errors.Wrap(err, "Could not query nameservers")
	}); err != nil {
		return nil, err
	}

	if r.Rcode != dns.RcodeSuccess {
		return nil, errors.New("Query was not successful")
	}

	for _, a := range r.Ns {
		if ns, ok := a.(*dns.NS); ok {
			var addr net.IP
			// We can't resolve the root nameserver names so store glue records
			for _, e := range r.Extra {
				if nsa, ok := e.(*dns.A); ok && nsa.Header().Name == ns.Ns {
					addr = nsa.A
					break
				}
			}

			if addr != nil {
				masters = append(masters, addr.String())
			}
		}
	}

	return masters, nil
}
