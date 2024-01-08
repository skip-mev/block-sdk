package types

const (
	DefaultEnabled = false
)

// NewParams returns a new Params instance with the provided values.
func NewParams(
	enabled bool,
) Params {
	return Params{
		Enabled: enabled,
	}
}

// DefaultParams returns the default x/blocksdk parameters.
func DefaultParams() Params {
	return NewParams(
		DefaultEnabled,
	)
}

// ValidateBasic performs basic validation on the parameters.
func (p *Params) ValidateBasic() error {
	return nil
}
