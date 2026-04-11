package validation

// CustomValidator allows external services to plug custom validation logic
// when the registry module is consumed as a library.
type CustomValidator interface {
	// Validate is invoked by the consuming service to run domain specific checks.
	// Implementations can decide what side effects or return handling they need.
	Validate()
}
