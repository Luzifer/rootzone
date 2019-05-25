package main

import (
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func getOpenNICTLDs() ([]string, error) {
	c := new(dns.Client)
	c.Net = "tcp"

	m := new(dns.Msg)
	m.SetQuestion("tlds.opennic.glue.", dns.TypeTXT)

	r, _, err := c.Exchange(m, cfg.OpenNICRoot+":53")
	if err != nil {
		return nil, errors.Wrap(err, "Could not query nameservers")
	}

	if r.Rcode != dns.RcodeSuccess {
		return nil, errors.New("Query was not successful")
	}

	tlds := []string{}

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

	r, _, err := c.Exchange(m, cfg.OpenNICRoot+":53")
	if err != nil {
		return nil, errors.Wrap(err, "Could not query nameservers")
	}

	if r.Rcode != dns.RcodeSuccess {
		return nil, errors.Errorf("Query was not successful: %s", dns.RcodeToString[r.Rcode])
	}

	masters := []string{"ns0.opennic.glue."}

	for _, a := range r.Answer {
		if cname, ok := a.(*dns.CNAME); ok {
			masters = append(masters, cname.Target)
		}
	}

	masterIPs := []string{}

	for _, master := range masters {
		m = new(dns.Msg)
		m.SetQuestion(master, dns.TypeA)

		r, _, err := c.Exchange(m, cfg.OpenNICRoot+":53")
		if err != nil {
			return nil, errors.Wrap(err, "Could not query nameservers")
		}

		if r.Rcode != dns.RcodeSuccess {
			return nil, errors.Errorf("Query was not successful: %s", dns.RcodeToString[r.Rcode])
		}

		for _, a := range r.Answer {
			if rr, ok := a.(*dns.A); ok {
				masterIPs = append(masterIPs, rr.A.String())
			}
		}
	}

	return masterIPs, nil
}
