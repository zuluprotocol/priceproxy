# Introduction

Thank you for considering contributing to `priceproxy`. It's much appreciated, and helps make the tool as useful as possible.

Following these guidelines helps to communicate that you respect the time of the developers managing and developing this open source project. In return, they should reciprocate that respect in addressing your issue, assessing changes, and helping you finalize your pull requests.

`priceproxy` is an open source project and we enjoy receiving contributions from our community. There are many ways to contribute, from writing tutorials or blog posts, improving the documentation, submitting bug reports and feature requests or writing code which can be incorporated into the tool itself.

Please don't use the issue tracker for support questions. Use the Vega Community or Discord instead - see the Community section below for links.

# Ground Rules

Responsibilities:

* Ensure cross-platform compatibility: Linux, MacOSX and Windows.
* Create an issue describing the problem / bug / feature request.
* Create a branch and pull request, and link it to one or more issues.
* Be welcoming to newcomers and encourage diverse new contributors from all backgrounds. See the [Python Community Code of Conduct](https://www.python.org/psf/codeofconduct/).

# Your First Contribution

Unsure where to begin contributing? You can start by looking through the issue list, or by submitting small pull requests that fix typos or improve documentation

Working on your first Pull Request? You can learn how from this *free* series, [How to Contribute to an Open Source Project on GitHub](https://egghead.io/series/how-to-contribute-to-an-open-source-project-on-github).

At this point, you're ready to make your changes! Feel free to ask for help; everyone is a beginner at first :smile_cat:

If a maintainer asks you to "rebase" your PR, they're saying that a lot of code has changed, and that you need to update your branch so it's easier to merge.

# Getting started

For something that is bigger than a one or two line fix:

1. Create your own fork of the code
1. Do the changes in your fork
1. If you like the change and think the project could use it:
   * Be sure you have followed the code style for the project.
   * Watch the [CI pipeline](https://github.com/vegaprotocol/priceproxy/actions) to make sure all tests pass.
   * Submit a pull request.

# How to report a security vulnerability

If you find a security vulnerability, do **NOT** open an issue. Email hi@vega.xyz instead, with "priceproxy security vulnerability" in the Subject line.

In order to determine whether you are dealing with a security issue, ask yourself these two questions:
* Can I access something that's not mine, or something I shouldn't have access to?
* Can I disable something for other people?

# How to report a bug

When filing an issue, make sure to answer these questions:

1. What version of Go are you using (go version)?
1. What operating system and processor architecture are you using?
1. What did you do?
1. What did you expect to see?
1. What did you see instead?

General questions related to the Go programming language should go to the `golang-nuts` mailing list instead of the issue tracker. The gophers there will answer or ask you to file an issue if you've tripped over a bug.

# How to suggest a feature or enhancement

Please look for an existing issue that matches your feature/enhacement. If you can't find one, open a new issue on our issues list on GitHub which describes the feature you would like to see, why you need it, and how it should work. There are bound to be others out there with similar needs.

# Code review process

The core team welcomes pull requests. PRs that pass the CI pipeline and conform to coding style rules are appreciated.

Style guides:
- Code should be run through `gofmt`. Many text editors and IDEs can do this automatically.
- Unit tests are in their own package, e.g. If `bar/foo.go` starts with `package bar` then `bar/foo_test.go` starts with `package bar_test` and can only use exported functions from the `bar` package.
- [Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

# Community

- [Vega Community - Price Proxy thread](https://community.vega.xyz/t/the-price-proxy/406)
- Discord
