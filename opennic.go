package main

import (
	"fmt"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
)

func getOpenNICTLDs() ([]string, error) {
	c := new(dns.Client)
	c.Net = "tcp"

	m := new(dns.Msg)
	m.SetQuestion("tlds.opennic.glue.", dns.TypeTXT)

	var (
		err error
		r   *dns.Msg
	)

	if err = retry(func() error {
		r, _, err = c.Exchange(m, cfg.OpenNICRoot+":53")
		return err //nolint:wrapcheck // wrapped on the outside
	}); err != nil {
		return nil, fmt.Errorf("querying nameservers: %w", err)
	}

	if r.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("query returned unexpected status: %s", dns.RcodeToString[r.Rcode])
	}

	var tlds []string
	for _, a := range r.Answer {
		if txt, ok := a.(*dns.TXT); ok {
			tlds = append(tlds, txt.Txt...)
		}
	}

	return tlds, nil
}

func getOpenNICZoneMasters(tld string) ([]string, error) {
	log.WithField("tld", tld).Trace("Getting zone masters")

	c := new(dns.Client)
	c.Net = "tcp"

	m := new(dns.Msg)
	m.SetQuestion(tld+"opennic.glue.", dns.TypeCNAME)

	var (
		err error
		r   *dns.Msg
	)

	if err = retry(func() error {
		r, _, err = c.Exchange(m, cfg.OpenNICRoot+":53")
		return err //nolint:wrapcheck // wrapped on the outside
	}); err != nil {
		return nil, fmt.Errorf("querying nameservers: %w", err)
	}

	if r.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("query returned unexpected status: %s", dns.RcodeToString[r.Rcode])
	}

	masters := []string{"ns0.opennic.glue."}

	for _, a := range r.Answer {
		if cname, ok := a.(*dns.CNAME); ok {
			masters = append(masters, cname.Target)
		}
	}

	var masterIPs []string
	for _, master := range masters {
		m = new(dns.Msg)
		m.SetQuestion(master, dns.TypeA)

		if err = retry(func() error {
			r, _, err = c.Exchange(m, cfg.OpenNICRoot+":53")
			return err //nolint:wrapcheck // wrapped on the outside
		}); err != nil {
			return nil, fmt.Errorf("querying nameservers: %w", err)
		}

		if r.Rcode != dns.RcodeSuccess {
			return nil, fmt.Errorf("query returned unexpected status: %s", dns.RcodeToString[r.Rcode])
		}

		for _, a := range r.Answer {
			if rr, ok := a.(*dns.A); ok {
				masterIPs = append(masterIPs, rr.A.String())
			}
		}
	}

	return masterIPs, nil
}
