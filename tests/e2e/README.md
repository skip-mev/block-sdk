# Block SDK E2E Tests

## Integrating/Running Tests In Your Project

The Block SDK E2E tests are built around [interchaintest](https://github.com/strangelove-ventures/interchaintest). To run the 
full set of predefined test cases using your chain you can follow the steps below:

### Integrate the Block SDK into your project  

You will need to pull in the Block SDK as a dependency of your project. You can follow the guides located in 
[our integration docs](https://docs.skip.money/chains/integrate-the-sdk/) or [this README](../../tests/app/README.md) for detailed instructions.  
You can refer to [this PR](https://github.com/CascadiaFoundation/cascadia/pull/33) as one example of how the Block SDK
can be integrated into an application. Be aware that each chain will have different requirements and this example may
not be fully applicable to yours. Additionally, future versions may introduce new requirements.

### Define a container image  

Interchaintest will require a Docker image in order to spin up your chain's application. If you do not already have a
container build setup that launches your application's binary you will need to define one.  
An example of the Block SDK's test app's Docker build can be referenced [here](../../contrib/images/block-sdk.e2e.Dockerfile).

### Create an InterchainTest Chain Spec  

The test suite uses the [interchaintest.ChainSpec](https://github.com/strangelove-ventures/interchaintest/blob/main/chainspec.go)
to define the specifics around running your application. You'll need to define the number of validator and full node containers to run, the docker image, the genesis state,
encoding config, and various other variables.  
Some examples of ChainSpecs follow:

* [Block SDK testappd](https://github.com/skip-mev/block-sdk/blob/335d7a216aaca757fee40a1ed63e13061e79b39d/tests/e2e/block_sdk_e2e_test.go#L71)
* [Cascadia](https://github.com/CascadiaFoundation/cascadia/blob/main/interchaintest/block_sdk_test.go)

### Instantiate the test suite and run the tests  

The following snippet is usually suitable.

```go
func TestBlockSDKSuite(t *testing.T) {
	s := integration.NewIntegrationTestSuiteFromSpec(spec)
	s.WithDenom(denom)
	s.WithKeyringOptions(encodingConfig.Codec, keyring.Option())
	suite.Run(t, s)
```

### Run the tests

Add a separate makefile target to run your tests.
