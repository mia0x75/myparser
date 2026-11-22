module github.com/mia0x75/myparser

go 1.27.3

require (
	github.com/mia0x75/parserkit v0.0.0
	golang.org/x/text v0.43.0
)

// Phase 3 local development: resolve parserkit from the sibling checkout.
replace github.com/mia0x75/parserkit => ../parserkit
