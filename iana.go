package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

const ianaRequestTimeout = 2 * time.Second

func getIANATLDs() (tlds []string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), ianaRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.IANATldList, nil)
	if err != nil {
		return nil, fmt.Errorf("creating IANA TLD request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("retrieving IANA TLD list: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.WithError(err).Error("closing IANA TLD request")
		}
	}()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if b := scanner.Bytes(); len(b) == 0 || b[0] == '#' {
			continue
		}

		tlds = append(tlds, strings.ToLower(scanner.Text())+".")
	}

	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning IANA TLD list: %w", err)
	}

	return tlds, nil
}

func getIANAZoneMasters(tld string) (masters []string, err error) {
	logrus.WithField("tld", tld).Trace("Getting zone masters")

	c := new(dns.Client)
	c.Net = "tcp"

	m := new(dns.Msg)
	m.SetQuestion(tld, dns.TypeNS)

	var r *dns.Msg
	if err = retry(func() error {
		rootServer, err := getRandomInternicRoot(context.Background())
		if err != nil {
			return fmt.Errorf("getting root-server to query: %w", err)
		}

		r, _, err = c.Exchange(m, rootServer)
		return err //nolint:wrapcheck // wrapped on the outside
	}); err != nil {
		return nil, fmt.Errorf("querying nameservers: %w", err)
	}

	if r.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("query returned unexpected status: %s", dns.RcodeToString[r.Rcode])
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
