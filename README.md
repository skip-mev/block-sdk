# ⚠️ Block SDK - No Longer Maintained ⚠️

🚨 **This project is no longer being maintained and will not be supported by current integrators.** If you wish to use this project, we strongly recommend forking and maintaining the code for any dependency or feature improvements you require. 🚨

---

## 🤔 What is the Block SDK?

**🌐 The Block SDK is a toolkit for building customized blocks.** The Block SDK is a set of Cosmos SDK and ABCI++ primitives that allow chains to fully customize blocks to specific use cases. It turns your chain's blocks into a **`highway`** consisting of individual **`lanes`** with their own special functionality.

Skip has built out a number of plug-and-play `lanes` on the SDK that your protocol can use, including in-protocol MEV recapture and Oracles! Additionally, the Block SDK can be extended to add **your own custom `lanes`** to configure your blocks to exactly fit your application needs.

## Release Compatibility Matrix

| Block SDK Version | Cosmos SDK |
| :---------: | :--------: |
|   `v1.x.x`    |  `v0.47.x`   |
|   `v2.x.x`    |  `v0.50.x`   |

## 📚 Block SDK Documentation

To read more about how the Block SDK works, check out the [How it Works](https://docs.skip.money/blocksdk/overview).

### 🏪 Lane App Store

To read more about Skip's pre-built `lanes` and how to use them, check out the [Lane App Store](https://docs.skip.money/blocksdk/lanes/existing-lanes/mev).

### 🎨 Lane Development

To read more about how to build your own custom `lanes`, check out the [Build Your Own Lane](https://docs.skip.money/blocksdk/lanes/build-your-own-lane).

## Audits 

The Block SDK has undergone audits by the following firms:

* [OtterSec (Sept 9th, 2023)](audits/ottersec_sept_9_2023.pdf): Post audit code released as [v1.2.0](https://github.com/skip-mev/block-sdk/releases/tag/v1.2.0) and [v2.0.0](https://github.com/skip-mev/block-sdk/releases/tag/v2.0.0) for the `v1.x.x` and `v2.x.x` release families, respectively.
