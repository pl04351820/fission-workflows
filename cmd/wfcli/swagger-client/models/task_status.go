// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
)

// TaskStatus task status
// swagger:model TaskStatus
type TaskStatus struct {

	// error
	Error *Error `json:"error,omitempty"`

	// fn ref
	FnRef *FnRef `json:"fnRef,omitempty"`

	// status
	Status TaskStatusStatus `json:"status,omitempty"`

	// updated at
	UpdatedAt strfmt.DateTime `json:"updatedAt,omitempty"`
}

// Validate validates this task status
func (m *TaskStatus) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateError(formats); err != nil {
		// prop
		res = append(res, err)
	}

	if err := m.validateFnRef(formats); err != nil {
		// prop
		res = append(res, err)
	}

	if err := m.validateStatus(formats); err != nil {
		// prop
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *TaskStatus) validateError(formats strfmt.Registry) error {

	if swag.IsZero(m.Error) { // not required
		return nil
	}

	if m.Error != nil {

		if err := m.Error.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("error")
			}
			return err
		}
	}

	return nil
}

func (m *TaskStatus) validateFnRef(formats strfmt.Registry) error {

	if swag.IsZero(m.FnRef) { // not required
		return nil
	}

	if m.FnRef != nil {

		if err := m.FnRef.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("fnRef")
			}
			return err
		}
	}

	return nil
}

func (m *TaskStatus) validateStatus(formats strfmt.Registry) error {

	if swag.IsZero(m.Status) { // not required
		return nil
	}

	if err := m.Status.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("status")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *TaskStatus) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *TaskStatus) UnmarshalBinary(b []byte) error {
	var res TaskStatus
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}