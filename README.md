github-release
======

[![Build Status](https://travis-ci.com/GabLeRoux/github-release.svg?branch=master)](https://travis-ci.com/GabLeRoux/github-release)
[![Docker Stars](https://img.shields.io/docker/stars/gableroux/github-release.svg)](https://hub.docker.com/r/gableroux/github-release)
[![Docker Pulls](https://img.shields.io/docker/pulls/gableroux/github-release.svg)](https://hub.docker.com/r/gableroux/github-release)
[![Image](https://images.microbadger.com/badges/image/gableroux/github-release.svg)](https://microbadger.com/images/gableroux/github-release)
[![Version](https://images.microbadger.com/badges/version/gableroux/github-release.svg)](https://microbadger.com/images/gableroux/github-release)
[![Layers](https://images.microbadger.com/badges/image/gableroux/github-release.svg)](https://microbadger.com/images/gableroux/github-release)

*This is a fork of [aktau/github-release](https://github.com/aktau/github-release) with
a few tweaks. All the original credit should go directly to Nicolas Hillegeer :)*

A small commandline app written in Go that allows you to easily create
and delete releases of your projects on Github. In addition it allows
you to attach files to those releases.

It interacts with the [github releases API](http://developer.github.com/v3/repos/releases).
Though it's entirely possibly to [do all these things with
cURL](https://github.com/blog/1645-releases-api-preview). It's not
really that user-friendly. For example, you need to first query the API
to find the id of the release you want, before you can upload an
artifact. `github-release` takes care of those little details.

It might still be a bit rough around the edges, pull requests are
welcome!

How to install
==============

If/when you have Go installed, you can just do:

```sh
go get github.com/gableroux/github-release
```

This will automatically download, compile and install the app.

After that you should have a `github-release` executable in your
`$GOPATH/bin`.

How to use
==========

**NOTE**: for these examples I've [created a github
token](https://help.github.com/articles/creating-an-access-token-for-command-line-use)
and set it as the env variable `GITHUB_TOKEN`. `github-release` will
automatically pick it up from the environment so that you don't have to
pass it as an argument.

```sh
# set your token
export GITHUB_TOKEN=...

# check the help
$ github-release --help

# make your tag and upload
$ git tag ... && git push --tags

# check the current tags and existing releases of the repo
$ github-release info -u gableroux -r gofinance
git tags:
- v0.1.0 (commit: https://api.github.com/repos/aktau/gofinance/commits/f562727ce83ce8971a8569a1879219e41d56a756)
releases:
- v0.1.0, name: 'hoary ungar', description: 'something something dark side 2', id: 166740, tagged: 29/01/2014 at 14:27, published: 30/01/2014 at 16:20, draft: ✔, prerelease: ✗
  - artifact: github.go, downloads: 0, state: uploaded, type: application/octet-stream, size: 1.9KB, id: 68616

# create a formal release
$ github-release release \
    --user gableroux \
    --repo gofinance \
    --tag v0.1.0 \
    --name "the wolf of source street" \
    --description "Not a movie, contrary to popular opinion. Still, my first release!" \
    --pre-release

# you've made a mistake, but you can edit the release without
# having to delete it first (this also means you can edit without having
# to upload your files again)
$ github-release edit \
    --user gableroux \
    --repo gofinance \
    --tag v0.1.0 \
    --name "Highlander II: The Quickening" \
    --description "This is the actual description!"

# upload a file, for example the OSX/AMD64 binary of gableroux's gofinance app
$ github-release upload \
    --user gableroux \
    --repo gofinance \
    --tag v0.1.0 \
    --name "gofinance-osx-amd64" \
    --file bin/darwin/amd64/gofinance

# upload other files...
$ github-release upload ...

# you're not happy with it, so delete it
$ github-release delete \
    --user gableroux \
    --repo gofinance \
    --tag v0.1.0
```

Sometimes (too often), Github uploads fail with HTTP code 422. You can use the
`-R` or `--replace` flag to delete any previous upload (failed or not) with the
exact same name in a release

```sh
# re-upload a file, we mean it this time!
$ github-release upload \
    --user gableroux \
    --repo gofinance \
    --tag v0.1.0 \
    --name "gofinance-osx-amd64" \
    --file bin/darwin/amd64/gofinance \
    --replace
```

GitHub Enterprise Support
=========================
You can point to a different GitHub API endpoint via the environment variable ```GITHUB_API```:

```sh
export GITHUB_API=http://github.company.com/api/v3
```

Used libraries
==============

| Package                                                                    | Description         | License |
| ------------------------------------------------------------------------   | ------------------- | ------- |
| [github.com/dustin/go-humanize](https://github.com/dustin/go-humanize)     | humanize file sizes | MIT     |
| [github.com/tomnomnom/linkheader](https://github.com/tomnomnom/linkheader) | GH API pagination   | MIT     |
| [github.com/voxelbrain/goptions](https://github.com/voxelbrain/goptions)   | option parsing      | BSD     |

Copyright
=========

Copyright (c) 2016, Amos Wenger & the itch.io team. All rights reserved.

Copyright (c) 2014, Nicolas Hillegeer. All rights reserved.

