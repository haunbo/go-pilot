package xecho

import (
	"errors"
	"reflect"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	translator "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
)

// Validator validates request structs and returns translated error messages.
// The "label" struct tag is used as the field name shown in errors.
type Validator struct {
	validate *validator.Validate
	trans    translator.Translator
}

// NewValidator creates a validator. locale defaults to "en"; pass "zh" for
// Simplified Chinese messages.
func NewValidator(locales ...string) *Validator {
	locale := "en"
	if len(locales) > 0 && locales[0] != "" {
		locale = locales[0]
	}

	v := validator.New()
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		if label := field.Tag.Get("label"); label != "" {
			return label
		}
		return field.Name
	})

	uni := translator.New(en.New(), zh.New())
	trans, found := uni.GetTranslator(locale)
	if !found {
		trans, _ = uni.GetTranslator("en")
	}
	enTrans, _ := uni.GetTranslator("en")
	zhTrans, _ := uni.GetTranslator("zh")
	_ = enTranslations.RegisterDefaultTranslations(v, enTrans)
	_ = zhTranslations.RegisterDefaultTranslations(v, zhTrans)
	return &Validator{validate: v, trans: trans}
}

// Validate implements echo.Validator.
func (v *Validator) Validate(i any) error {
	err := v.validate.Struct(i)
	if err == nil {
		return nil
	}
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) && len(validationErrors) > 0 {
		return errors.New(validationErrors[0].Translate(v.trans))
	}
	return err
}
