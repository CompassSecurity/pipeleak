# Pipeleak

Scan GitLab job output logs for secrets

# Get Started

Download the binary from the [Releases](https://github.com/CompassSecurity/pipeleak/releases) page.

```bash
pipeleak scan --token glpat-xxxxxxxxxxx --gitlab https://gitlab.com
```

# Rules Tweaking

Upon the first run the [file](https://github.com/mazen160/secrets-patterns-db/blob/master/db/rules-stable.yml) `rules.yml` is created. 
To remove or add rules, just adapt this file.