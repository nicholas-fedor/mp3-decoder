# AGENTS.md

Instructions for coding agents in this repository.

## Project

mp3-decoder is a pure-Go decoder for MPEG-1, MPEG-2 and MPEG-2.5 audio in Layers I, II and III. Its root package is named `mp3`. [CONTRIBUTING.md](CONTRIBUTING.md) holds the setup, the gate, the decoder rules, the code style and the test layout. Read it before you change anything. Everything in it applies to you.

## Workflow

- Never commit, push, tag or rewrite history without the maintainer's approval for that action. Approval of one commit does not extend to the next.
- While you work, run the tests and lint of the packages you touch, such as `go test -race -count=1 ./internal/layer3/...` and `golangci-lint run --config build/golangci-lint/golangci-lint.yaml ./internal/layer3/...`.
- Run `task check` before a change that reaches the Taskfile, the lint config, `go.mod`, the decoder's output or packages across the tree.
- `issues.fix` is on, so golangci-lint can rewrite code. Review `git diff` after every lint run.
- Check how another decoder behaves in its source code, never from memory. The leading decoders are mpg123 and FFmpeg.

## Decoder Rules

The [Decoder Rules](CONTRIBUTING.md#decoder-rules) are the ones a change most often breaks. Check every change against each of them before you present it. Then check the [Tests](CONTRIBUTING.md#tests) and [Documentation](CONTRIBUTING.md#documentation) rules.

## Writing

These rules cover code comments, docs, commit messages and pull requests. The [Code Style](CONTRIBUTING.md#code-style) and [Documentation](CONTRIBUTING.md#documentation) rules apply too.

- No em dashes. No semicolons joining clauses. No chains of three or more clauses joined by commas. Write short sentences or a list.
- A colon only introduces a list.
- Doc comments hold no status, plans or test notes.
