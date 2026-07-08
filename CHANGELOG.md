# Changelog

## Unreleased

- Add a "Fail early" option to the service status check. When enabled (the default, matching the previous behavior), the "All the time" mode fails as soon as a deviating status is observed. When disabled, the check keeps collecting events for the whole duration and only fails at the end of the step (with a past-tense message, since the status may have recovered by then). Only affects the "All the time" mode.

## v1.0.28

- chore(deps): bump github.com/steadybit/action-kit/go/action_kit_sdk
- chore(deps): bump github.com/steadybit/discovery-kit/go/discovery_kit_sdk
- chore(deps): bump github.com/steadybit/extension-kit
- chore: add Claude Code workflows (#144)
- chore: silence SonarQube finding on secrets: inherit in Claude workflows
- fix: escape the service id used in the StackState snapshot query and build the request body with a JSON encoder, preventing STQL/JSON query injection
- fix: guard the service check and discovery against missing components, identifiers, short base URLs and unexpected identifier formats instead of panicking, and avoid a possible nil-dereference when a StackState request fails before a response is received
- fix: prevent STQL injection and reachable panics in service check/discovery (#145)

## v1.0.27

- chore(deps): bump github.com/steadybit/extension-kit

## v1.0.26

- chore(deps): bump golang.org/x/net to v0.55.0 (CVE-2026-39821) (#140)

## v1.0.25

- chore(deps): bump alpine from 3.23 to 3.24

## v1.0.24

- chore: update to go 1.26.4
- feat: add weekly auto patch-release workflow

## v1.0.23

- Support discovery group attribute via `STEADYBIT_EXTENSION_DISCOVERY_GROUP` env var (or `discovery.group` Helm value) — when set, the extension adds `steadybit.group=<value>` to every discovered target
- Update dependencies

## v1.0.22

- Bump Go to 1.26.3
- Update dependencies

## v1.0.21

- Bump Go to 1.25.9
- Support if-none-match for the extension list endpoint
- Update dependencies

## v1.0.20

- feat(chart): split image.name into image.registry + image.name
- Support global.priorityClassName
- Update alpine packages in Docker image to address CVEs
- Update dependencies

## v1.0.19

- Update dependencies

## v1.0.18

- Update dependencies

## v1.0.17

- Update dependencies

## v1.0.16

- Updated dependencies

## v1.0.15

- Update dependencies

## v1.0.14

- Provide service status check mode to verify if the given state was observed at least once or all the time.

## v1.0.13

- Fix missing property in StackState API request issue.
- Update dependencies

## v1.0.12

- Use uid instead of name for user statement in Dockerfile
- Update dependencies

## v1.0.11

- Set new `Technology` property in extension description
- Update dependencies (go 1.23)

## v1.0.10

- Update dependencies (go 1.22)

## v1.0.9

- Update dependencies

## v1.0.8

- Update dependencies

## v1.0.7

- Update dependencies

## v1.0.6

- Update dependencies

## v1.0.5

- Added `pprof` endpoints for debugging purposes
- Update dependencies

## v1.0.4

- Possibility to exclude attributes from discovery

## v1.0.3

- algin the attribute names for kubernetes objects

## v1.0.2

- update dependencies

## v1.0.1

 - migration to new unified steadybit actionIds and targetTypes

## v1.0.0

 - Initial release
