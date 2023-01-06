package nftModels

import (
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/kube-openapi/pkg/validation/validate"
)

type AssetAttribute struct {
	// maximum value
	// Example: 40
	// Required: true
	MaxValue *int64 `json:"maxValue"`

	// name
	// Example: ratio
	// Required: true
	Name *string `json:"trait_type"`

	// value
	// Example: 20
	// Required: true
	Value *int64 `json:"value"`
}

// Validate validates this asset level
func (a *AssetAttribute) Validate() error {
	var res []error

	if err := a.validateMaxValue(); err != nil {
		res = append(res, err)
	}

	if err := a.validateName(); err != nil {
		res = append(res, err)
	}

	if err := a.validateValue(); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		err := fmt.Sprintln(res)
		return errors.New(err)
	}
	return nil
}

func (a *AssetAttribute) ValidateValue() error {
	var res []error

	if err := a.validateName(); err != nil {
		res = append(res, err)
	}

	if err := a.validateValue(); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		err := fmt.Sprintln(res)
		return errors.New(err)
	}
	return nil
}

func (a *AssetAttribute) validateMaxValue() error {

	if err := validate.Required("maxValue", "attributes", a.MaxValue); err != nil {
		return err
	}

	return nil
}

func (a *AssetAttribute) validateName() error {

	if err := validate.Required("name", "attributes", a.Name); err != nil {
		return err
	}

	return nil
}

func (a *AssetAttribute) validateValue() error {

	if err := validate.Required("value", "attributes", a.Value); err != nil {
		return err
	}

	return nil
}
