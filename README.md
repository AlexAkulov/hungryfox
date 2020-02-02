# HungryFox

[![Build Status](https://travis-ci.org/AlexAkulov/hungryfox.svg?branch=master)](https://travis-ci.org/AlexAkulov/hungryfox)
[![codecov](https://codecov.io/gh/AlexAkulov/hungryfox/branch/master/graph/badge.svg)](https://codecov.io/gh/AlexAkulov/hungryfox)


**State: In development now! You probably will get many bugs!**

HungryFox is a continuous search tool. It scans git repositories for leaks of sensitive information like passwords, api-keys, private certificates, etc. As an experimental feature, hungryfox supports scanning dependencies and searching for related software vulnerabilities.

HungryFox differs from other solutions as it can work as a daemon and efficiently scans each new commit in repo and sends notification about found leaks.

HungryFox uses regex-patterns to search for vulnerabilities.
 
It is hard to write a good enough regex-pattern that could simultaneously find all leaks without generating a lot of false positive events so HungryFox in addition to regex-patterns has regex-filters. You can write a weak regex-pattern for search leaks and skip known false positive with the help of regex-filters.

Hungryfox also supports filtering false positives by entropy. Use Entropies.WordMin and Entropies.LineMin options in pattern configuration to filter out all leaks with lesser entropy. Word entropy is calculated as the largest Shannon entropy of words in a line, whereas line entropy is computed from the whole line. A leak is considered a false positive if it's less than both WordMin and LineMin. Experiments show that setting both to 3.0 safely cuts off some false positives. Higher values like 3.2, 3.5 filter much more false positives, but occasionally filter out real passwords.


## Features
- [x] Patterns and filters
- [x] Entropy filtering
- [x] State support
- [x] Notifications by email
- [x] History limit by time
- [x] GitHub support
- [x] Gitlab API support
- [ ] Written in pure go, does not require external git ([wait](https://github.com/src-d/go-git/issues/757))
- [ ] Line number of leak ([wait](https://github.com/src-d/go-git/issues/806))
- [ ] Custom email templates
- [ ] GitHook support
- [ ] HTTP Api
- [ ] WebUI
- [ ] Tests
- [ ] Integration with Hashicorp Vault

**Experimental features:**
- [x] Finding dependencies in .csproj files
- [ ] Finding dependencies in NPM packages
- [x] Searching for vulnerabilities in [OSS Index database](https://ossindex.sonatype.org/)
- [ ] Searching for vulnerabilities in [NVD](https://nvd.nist.gov/)

## Installation

### From Sources

```
go get github.com/AlexAkulov/hungryfox/cmd/hungryfox
```

### From [packagecloud.io](https://packagecloud.io/AlexAkulov/hungryfox-unstable)

[![](https://img.shields.io/badge/deb-packagecloud.io-844fec.svg)](https://packagecloud.io/AlexAkulov/hungryfox-unstable/install#bash-deb)
[![](https://img.shields.io/badge/rpm-packagecloud.io-844fec.svg)](https://packagecloud.io/AlexAkulov/hungryfox-unstable/install#bash-rpm)


## Configuration
```
common:
  state_file: /var/lib/hungryfox/state.yml
  history_limit: 1y
  scan_interval: 30m
  workers: 4
  leaks_file: /var/lib/hungryfox/leaks.json
  enable_leaks_scanner: true                    # true by default
  enable_exposures_scanner: false               # false by default
  
logging:
  level: debug
  file: /var/lib/hungryfox/logs/log             # optional, log to rolling file, logs to console by default

smtp:
  enable: true
  host: smtp.kontur
  port: 25
  username: smtpUser
  password: smtpPassword
  mail_from: hungryfox@example.com
  disable_tls: true
  recipient: security@example.com               # auditor's email, that receives all letters
  sent_to_author: false                         # send leak and vulnerability letters to commit authors
  recipient_regex: @yourorganization\.com$      # optinal, only send a letter, if the recipent's email matches

webhook:
  enable: true
  method: POST
  url: https://example.com/webhook
  headers:
    x-sample-header: value
    
exposures:
  oss_index_user: foo
  oss_index_password: bar
  suppressions_path: /var/lib/hungryfox/settings/suppressions.json      # optional, suppressions for exposures scanner
  exposures_file: /var/lib/hungryfox/exposures.json                     # writes exposures to this file

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
  # Inspects for leaks on Gitlab. Uses Gitlab API to list projects, then scans matching repositories.
  - type: gitlab
    token: # required to access gitlab api
    work_dir: "/var/hungryfox/gitlab"
    gitlab_url: https://gitlab.org.com
    gitlab_exclude_namespaces: # these project groups won't be scanned
      - group1
    gitlab_exclude_projects: # these projects won't be scanned
      - project 1
    gitlab_filter:  # a search string to pass to gitlab API when fetching projects list
    gitlab_include_non_group: false # whether to scan private repositories

patterns:
  - name: secret in my code                 # not required
    file: \.go$                             # .+ by default
    content: (?i)secret = ".+"              # .+ by default
    entropies:                              # not required
      word_min: 3.0                         # see above
      line_min: 3.0

filters:
  - name: skip any leaks in tests           # not required
    file: /IntegrationTests/.+_test\.go$    # .+ by default
    # content:                              # .+ by default
```
## Performance
We use HungryFox for scanning ~3,5K repositories on our GitLab server and about one hundred repositories on GitHub

## Alternatives
- [Gitrob by michenriksen](https://github.com/michenriksen/gitrob)
- [Gitleaks by zricethezav](https://github.com/zricethezav/gitleaks)
- [git-secrets by AWSLabs](https://github.com/awslabs/git-secrets)
- [Truffle Hog by dxa4481](https://github.com/dxa4481/truffleHog)
- [repo-scraper by dssg](https://github.com/dssg/repo-scraper)
- [Security Scan by onetwopunch](https://github.com/onetwopunch/security-scan)
- [repo-security-scanner by UKHomeOffice](https://github.com/UKHomeOffice/repo-security-scanner)
- [detect-secrets by Yelp](https://github.com/Yelp/detect-secrets)
- [Github Dorks by techgaun](https://github.com/techgaun/github-dorks)
- [Repo Supervisor by Auth0](https://github.com/auth0/repo-supervisor)
- [git-all-secrets by anshumanbh](https://github.com/anshumanbh/git-all-secrets)
