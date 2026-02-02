Git hooks used by this repo.

Enable hooks locally:
  make hooks

This sets:
  git config core.hooksPath .githooks

Note: Git does NOT run hooks automatically from the repository unless core.hooksPath is configured.
