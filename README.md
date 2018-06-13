# HungryFox

[![Build Status](https://travis-ci.org/AlexAkulov/hungryfox.svg?branch=master)](https://travis-ci.org/AlexAkulov/hungryfox)
[![codecov](https://codecov.io/gh/AlexAkulov/hungryfox/branch/master/graph/badge.svg)](https://codecov.io/gh/AlexAkulov/hungryfox)
[![Known Vulnerabilities](https://snyk.io/test/github/AlexAkulov/hungryfox/badge.svg)](https://snyk.io/test/github/AlexAkulov/hungryfox)

**State: In development now! You probably will get many bugs!**

HungryFox is a software for continious search for leaks of sensative information like passwords, api-keys, private certificates and etc in your repositories.

HungryFox differs from other solutions as it can work as a daemon and efficiently scans each new commit in repo and sends notification about found leaks.

HungryFor works on regex-patterns only and doesn't use analyze by entropy because in my opinion this way generates a lot of false positive events. Maybe analyse by entropy will be added in future.

It is hard to write a good enough regex-pattern that could simultaneously find all leaks and not to generate a lot of false positive events so HungryFox in addition with regex-patterns has regex-filtres. You can write weak regex-pattern for search leaks and skip known false positive with the help of regex-filters.


## Features
- [x] Patterns and filtres
- [x] State support
- [x] Notifications by email
- [x] History limit by time
- [x] GitHub-support
- [ ] Written on pure go and no requirement of external git ([wait](https://github.com/src-d/go-git/issues/757))
- [ ] Line number of leak ([wait](https://github.com/src-d/go-git/issues/806))
- [ ] GitHook support
- [ ] HTTP Api
- [ ] WebUI
- [ ] Tests
- [ ] Integration with Hashicorp Vault

## Installation

```
go get github.com/AlexAkulov/hungryfox
```

## Configuation
```
common:
  state_file: /var/lib/hungryfox/state.yml
  history_limit: 1y
  scan_interval: 30m
  log_level: debug
  leaks_file: /var/lib/hungryfox/leaks.json

smtp:
  enable: true
  host: smtp.kontur
  port: 25
  mail_from: hungryfox@example.com
  disable_tls: true
  recipient: security@example.com
  sent_to_author: false

inspect:
  # Inspects for leaks in your local repositories without clone or fetch. It is suitable for running on git-server
  - type: path
    trim_prefix: "/var/volume/repositories"
    trim_suffix: ".git"
    url: https://gitlab.example.com
    paths:
      - "/data/gitlab/repositories/*/*.git"
      - "/data/gitlab/repositories/*/*/*.git"
      - "!/data/gitlab/repositories/excluded/repo.git"
  # Inspects for leaks on GitHub. HungryFox will clone the repositories into work_dir and fetch them before scannig
  - type: github
    token: # is required for scanning private repositories
    work_dir: "/var/hungryfox/github"
    users:
      - AlexAkulov
    repos:
      - moira-alert/moira
    orgs:
      - skbkontur

patterns:
  - name: secret in my code                 # not required
    file: \.go$                             # .+ by default
    content: (?i)secret = ".+"              # .+ by default

filters:
  - name: skip any leaks in tests           # not required
    file: /IntegrationTests/.+_test\.go$    # .+ by default
    # content:                              # .+ by default
```
## Perfomance
We use HungryFox for scanning ~3,5K repositories on our GitLab server and about one hundred repositories on GitHub

## Alternatives
- [Gitrob](https://github.com/michenriksen/gitrob)
- [Gitleaks](https://github.com/zricethezav/gitleaks)
- [git-secrets](https://github.com/awslabs/git-secrets)
- [Truffle Hog](https://github.com/dxa4481/truffleHog)
- [repo-scraper](https://github.com/dssg/repo-scraper)
- [Security Scan](https://github.com/onetwopunch/security-scan)
