package validators

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var (
	mobileRegex = regexp.MustCompile(`^\+[0-9]{12}$`)
)

// RegisterCustomValidators registers all custom validators with Gin's validator
func RegisterCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("mobile_number", validateMobileNumber)
		// Register other custom validators here as needed
		// v.RegisterValidation("custom_validator", customValidatorFunc)
	}
}

// validateMobileNumber validates mobile number format
func validateMobileNumber(fl validator.FieldLevel) bool {
	return mobileRegex.MatchString(fl.Field().String())
}
