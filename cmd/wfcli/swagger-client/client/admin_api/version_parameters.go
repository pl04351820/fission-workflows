// Code generated by go-swagger; DO NOT EDIT.

package admin_api

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

// NewVersionParams creates a new VersionParams object
// with the default values initialized.
func NewVersionParams() *VersionParams {

	return &VersionParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewVersionParamsWithTimeout creates a new VersionParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewVersionParamsWithTimeout(timeout time.Duration) *VersionParams {

	return &VersionParams{

		timeout: timeout,
	}
}

// NewVersionParamsWithContext creates a new VersionParams object
// with the default values initialized, and the ability to set a context for a request
func NewVersionParamsWithContext(ctx context.Context) *VersionParams {

	return &VersionParams{

		Context: ctx,
	}
}

// NewVersionParamsWithHTTPClient creates a new VersionParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewVersionParamsWithHTTPClient(client *http.Client) *VersionParams {

	return &VersionParams{
		HTTPClient: client,
	}
}

/*VersionParams contains all the parameters to send to the API endpoint
for the version operation typically these are written to a http.Request
*/
type VersionParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the version params
func (o *VersionParams) WithTimeout(timeout time.Duration) *VersionParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the version params
func (o *VersionParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the version params
func (o *VersionParams) WithContext(ctx context.Context) *VersionParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the version params
func (o *VersionParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the version params
func (o *VersionParams) WithHTTPClient(client *http.Client) *VersionParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the version params
func (o *VersionParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *VersionParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
