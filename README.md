<h1 align="center">Block SDK 🧱</h1>

<!-- markdownlint-disable MD013 -->
<!-- markdownlint-disable MD041 -->
[![Project Status: Active – The project has reached a stable, usable state and is being actively developed.](https://www.repostatus.org/badges/latest/active.svg)](https://www.repostatus.org/#wip)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue?style=flat-square&logo=go)](https://godoc.org/github.com/skip-mev/block-sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/skip-mev/block-sdk?style=flat-square)](https://goreportcard.com/report/github.com/skip-mev/block-sdk)
[![Version](https://img.shields.io/github/tag/skip-mev/block-sdk.svg?style=flat-square)](https://github.com/skip-mev/block-sdk/releases/latest)
[![License: Apache-2.0](https://img.shields.io/github/license/skip-mev/block-sdk.svg?style=flat-square)](https://github.com/skip-mev/block-sdk/blob/main/LICENSE)
[![Lines Of Code](https://img.shields.io/tokei/lines/github/skip-mev/block-sdk?style=flat-square)](https://github.com/skip-mev/block-sdk)

### 🤔 What is the Block SDK?

**🌐 The Block SDK is a toolkit for building customized blocks**. The Block SDK is a set of Cosmos SDK and ABCI++ primitives that allow chains to fully customize blocks to specific use cases. It turns your chain's blocks into a **`highway`** consisting of individual **`lanes`** with their own special functionality.


Skip has built out a number of plug-and-play `lanes` on the SDK that your protocol can use, including in-protocol MEV recapture and Oracles! Additionally, the Block SDK can be extended to add **your own custom `lanes`** to configure your blocks to exactly fit your application needs.

### 📚 Block SDK Documentation

To read more about how the Block SDK works, check out the [How it Works](https://docs.skip.money/chains/overview).

#### Lane "App Store"

To read more about Skip's pre-built `lanes` and how to use them, check out the [Lane App Store](https://docs.skip.money/chains/lanes/existing-lanes/default).

#### Lane Development

To read more about how to build your own custom `lanes`, check out the [Build Your Own Lane](https://docs.skip.money/chains/lanes/build-your-own-lane).
