[![Go Report Card](https://goreportcard.com/badge/github.com/Luzifer/rootzone)](https://goreportcard.com/report/github.com/Luzifer/rootzone)
![](https://badges.fyi/github/license/Luzifer/rootzone)
![](https://badges.fyi/github/downloads/Luzifer/rootzone)
![](https://badges.fyi/github/latest-release/Luzifer/rootzone)

# Luzifer / rootzone

`rootzone` is a small util for my [personal-dns](https://github.com/luzifer-docker/personal-dns) project to collect all IANA and OpenNIC TLDs and generate a named stub file for bind to be able to resolve those TLDs without delegation to third-party nameservers which might be modifying the original responses from the root nameservers.
