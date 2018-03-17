// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/validate"
)

// WorkflowSpecTasks Tasks contains the specs of the tasks, with the key being the task id.
//
// Note: Dependency graph is build into the tasks.
// swagger:model workflowSpecTasks
type WorkflowSpecTasks map[string]TaskSpec

// Validate validates this workflow spec tasks
func (m WorkflowSpecTasks) Validate(formats strfmt.Registry) error {
	var res []error

	if err := validate.Required("", "body", WorkflowSpecTasks(m)); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
