// Code generated by go-swagger; DO NOT EDIT.

package workflow_api

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"

	strfmt "github.com/go-openapi/strfmt"
)

// NewWfListParams creates a new WfListParams object
// with the default values initialized.
func NewWfListParams() *WfListParams {

	return &WfListParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewWfListParamsWithTimeout creates a new WfListParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewWfListParamsWithTimeout(timeout time.Duration) *WfListParams {

	return &WfListParams{

		timeout: timeout,
	}
}

// NewWfListParamsWithContext creates a new WfListParams object
// with the default values initialized, and the ability to set a context for a request
func NewWfListParamsWithContext(ctx context.Context) *WfListParams {

	return &WfListParams{

		Context: ctx,
	}
}

// NewWfListParamsWithHTTPClient creates a new WfListParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewWfListParamsWithHTTPClient(client *http.Client) *WfListParams {

	return &WfListParams{
		HTTPClient: client,
	}
}

/*WfListParams contains all the parameters to send to the API endpoint
for the wf list operation typically these are written to a http.Request
*/
type WfListParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the wf list params
func (o *WfListParams) WithTimeout(timeout time.Duration) *WfListParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the wf list params
func (o *WfListParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the wf list params
func (o *WfListParams) WithContext(ctx context.Context) *WfListParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the wf list params
func (o *WfListParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the wf list params
func (o *WfListParams) WithHTTPClient(client *http.Client) *WfListParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the wf list params
func (o *WfListParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *WfListParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
