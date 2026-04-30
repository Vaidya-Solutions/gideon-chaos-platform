# Gideon Chaos Platform

Grouped runtime boundary for Gideon's chaos and resilience-testing platform.

This grouped boundary is being bootstrapped as the runtime and platform-code
owner for chaos validation. The current `lambda-chaos-machine` runtime now
lives under this grouped root as the first moved child, while hosted-repo
cutover and later component splits remain deferred.

Later children may include:

- `chaos-steady-state`
- `chaos-start-experiment`
- `chaos-continue-execution`
- `chaos-evaluate-hypothesis`

Optional later surfaces such as a scenario catalog, evidence export, or
stack-specific probes remain deferred until real follow-on work justifies them.

## Status

This grouped boundary is in its initial runtime-host state.

What is complete in this slice:

- the grouped staging root now exists at `services/gideon-chaos-platform`
- the current `lambda-chaos-machine` runtime now lives under this grouped root
- grouped root ignore and context-exclusion files now exist
- grouped root shared lambda rules now exist under `makefiles/`
- grouped workflow ownership notes now exist under `.github/README.md`
- the active extraction checklist now tracks this grouped boundary directly
- a wrapper-only grouped root `Makefile` now exists
- an initial grouped workflow package now exists under `.github/workflows/`

What is not complete yet:

- no hosted repo boundary has been created yet
- no smaller child runtime directories have been created yet
- hosted-repo cutover remains deferred until the grouped layout validates
- the next implementation step is moving from thin command-shell boundaries to
  actual per-component runtime directories only where the package, test, and
  ARM64 build contract stays green

## Ownership Boundary

This grouped runtime boundary is intended to own:

- runtime code
- tests
- package and build contracts
- service-local `Makefile` surfaces
- grouped runtime workflow packaging
- scenario and runtime docs needed to build and validate the chaos platform

This grouped runtime boundary is not intended to own:

- Terraform modules or stacks
- ADRs
- runbooks
- governance docs
- policy or instruction sources
- environment promotion
- broader orchestration that remains above the runtime boundary

Those parent-owned surfaces remain in the Gideon meta repo by design.

## Commands

From the grouped root:

- `make help`
- `make test`
- `make lint`
- `make package-arm64`
- `make verify-all`
- `make clean`

The grouped root `Makefile` is wrapper-only and delegates to the direct-child
runtime layout now hosted under this grouped root.

## GitHub Automation

The grouped workflow package now exists under `.github/workflows/`.

Those workflows now match the local grouped repo-root layout where
`lambda-chaos-machine/` is a direct child of this grouped root, and they are
also the intended future-hosted workflow package for the extracted grouped
repo boundary.

## Layout Direction

The grouped layout direction is intentionally narrow:

- no nested `services/` directory inside `gideon-chaos-platform`
- no generic shared chaos framework introduced just to avoid small duplication
- no runtime split in the same slice as the grouped-boundary bootstrap

The first runtime move now preserves the current `lambda-chaos-machine`
contract under the grouped root. The grouped split-prep phase is now complete:
the four command entrypoints delegate to internal packages, which makes the
runtime boundaries explicit before any later directory-level component split.
