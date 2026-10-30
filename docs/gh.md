# GitHub from the terminal

[gh](https://cli.github.com) is GitHub's CLI: what you'd click on github.com,
typed instead. It's yours rather than the project's, logged in to your account
like git, so mise.toml doesn't pin it. Install it, then log in once:

```sh
gh auth login
```

## Pull requests

```sh
gh pr create --fill   # from your pushed branch, its commits as the text
gh pr checks --watch  # follow its CI to the end
gh pr view --web      # open it on github.com
```

## CI and releases

```sh
gh run list               # the latest workflow runs
gh run watch              # follow one while it runs, the release after a merge
gh run view --log-failed  # the logs of a run's failed steps
```

## Deploy settings

The README's [deploy](../README.md#deploying) step 3, without the settings page:

```sh
gh variable set DEPLOY_HOST --body user@host
gh secret set DEPLOY_SSH_KEY < ~/.ssh/fullmono_deploy
ssh-keyscan host | gh secret set DEPLOY_KNOWN_HOSTS
```

A variable can be read back; a secret can't, not even by you, only by the
workflows that use it.
