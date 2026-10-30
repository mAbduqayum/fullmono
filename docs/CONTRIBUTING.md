# Contributing

How a change gets from your machine into `main`. The [README](../README.md)
sets the project up, [architecture.md](architecture.md) shows how its parts
fit, and [gh.md](gh.md) does the GitHub steps from the terminal.

## Make a change

1. Branch off `main`:

    ```sh
    git switch main && git pull
    git switch -c fix/count-clicks
    ```

2. Run the stack with reload while you work: `just dev`.
3. Change the tests beside the code you touch:
    - `just test` runs the unit tests, in seconds;
    - `just test-integration` runs the ones against Postgres.
4. Commit, push, and open a pull request.

A schema change is a new migration in `apps/api/migrations/`, numbered after
the last one. Update [schema.sql](../apps/api/migrations/schema.sql) by hand
to match.

## Commit messages

Messages follow [Conventional Commits](https://www.conventionalcommits.org):
a type, an optional scope in brackets, then a summary.

```text
feat(web): show each link's click count
fix(api): answer 404 for an unknown code
docs: explain the deploy job
```

- **Type**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`,
  `build`, `ci`, `chore` or `revert`.
- **Scope**: the part it changes, `api`, `web` or `e2e`. Leave it out when a
  change spans them.
- **Summary**: in the imperative ("add", not "added"), lowercase, no full stop,
  short enough for `git log --oneline`.

## The git hooks

`just setup` installs them ([lefthook.yml](../lefthook.yml)):

| When         | What runs                                                  |
| ------------ | ---------------------------------------------------------- |
| `git commit` | the staged files formatted and linted; the message checked |
| `git push`   | the unit tests                                             |

> A failing hook stops the commit or the push. Fix what it reports instead of
> skipping it with `--no-verify`: CI runs the same lint and tests, so skipping
> only moves the failure there.

## Pull requests

A new pull request's text starts as the
[template](../.github/pull_request_template.md): what it changes and why, how
it was tested, and a checklist to tick before asking for a review.
[CI](../.github/workflows/ci.yml) checks every pull request, and only the apps
it changes.

A merge into `main` releases: [release.yml](../.github/workflows/release.yml)
runs CI again, pushes the images, then deploys them.
