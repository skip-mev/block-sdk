package keeper

// FrontRunningError defines a custom error type for detecting front-running or sandwich attacks.
type FrontRunningError struct{}

func NewFrontRunningError() *FrontRunningError {
	return &FrontRunningError{}
}

func (e FrontRunningError) Error() string {
	return "bundle contains transactions signed by multiple parties; possible front-running or sandwich attack"
}
