# CLAUDE.md

## How we work: one step at a time

This project is built incrementally, one function or snippet at a time, together with the user.

- At each step, propose **only the next small step**: what it is, why it comes next, and the code or change involved.
- **Stop and wait for the user to confirm** before doing anything else. Do not continue to the following step, and do not batch several steps together.
- Keep each step small enough for the user to read, understand, and run on their own.
- After a step is applied, say how to verify it (e.g. a `go build`, a test, or a `grpcurl` call), then propose the next step and wait again.
