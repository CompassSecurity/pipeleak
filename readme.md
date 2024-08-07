# Pipeleak

Scan GitLab job output logs for secrets

# Get Started

Download the binary from the [Releases](https://github.com/CompassSecurity/pipeleak/releases) page.

```bash
pipeleak scan --token glpat-xxxxxxxxxxx --gitlab https://gitlab.com
```

## Artifacts

Some pipelines generate artifacts which can be scanned as well.

```bash
pipeleak scan --token glpat-xxxxxxxxxxx --gitlab https://gitlab.com -a -c 
```

**Pro tip:**

> Dotenv artifacts are not available over the API. To scan these you must provide your session cookie value after a successful login in your browser. The cookie name is: `_gitlab_session`

```bash
pipeleak scan --token glpat-xxxxxxxxxxx --gitlab https://gitlab.com -v -a -c [value-of-valid-_gitlab_session]
```

# Rules Tweaking

Upon the first run the [file](https://github.com/mazen160/secrets-patterns-db/blob/master/db/rules-stable.yml) `rules.yml` is created. 
To remove or add rules, just adapt this file.