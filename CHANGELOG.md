# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) with the minor change that we use a prefix instead of grouping.
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.11.0] - 2024-01-11
- Fixed: binary build on ubuntu 22.04 doesn't run on ubuntu 22.04

## [1.10.0] - 2024-01-10
- Security: updated the dependencies in Github actions
- Added: Github action to have an automatically dependency check
- Fixed: make dist-check was skipped because of missing branch variable
- Fixed: added filename with the build folder
- Changed: moved away from deprecated Github actions

## [1.9.0] - 2024-01-09
- Changed: updated makefile to a more generic one
- Fixed: minor issues stated from the new make update, eg replaced deprecated ioutils

## [1.8.0] - 2024-01-09
- Removed: dependabot settings
- Security: updated dependencies

## [1.7.0] - 2023-07-17
- Security: updated dependencies

## [1.6.0] - 2023-04-11
- Security: updated dependencies
- Added: GitHub dependabot as a Test
- Changed: make update-dependencies to include test dependencies

## [1.5.0] - 2023-01-09
- Security: updated dependencies

## [1.4.0] - 2022-10-11
- Added: makefile support for update dependencies
- Added: makefile support for MacOs M1
- Fixed: issue in Make file where version number was not taken into account with compiled version
- Changed: changed release steps to use recommended repo as stated in https://github.com/actions/upload-release-asset#readme
- Security: updated dependencies

## [1.3.0] - 2022-07-11
- Security: updated dependencies

## [1.2.0] - 2022-04-22
- Security: updated dependencies

## [1.1.0] - 2021-12-14
- Added: access logs in Apache Common Log Format
- Added: an option to add metric labels to a probe config
- Added: response is compressed if requested
- Fixed: invalid path now gives a 404 error instead of the landingspage
- Changed: split main.go into multiple files
- Security: updated dependencies

## [1.0.0] - 2020-07-08
- Added: initial version of script exporter
