// Code generated by go-swagger; DO NOT EDIT.

package workflow_invocation_api

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/fission/fission-workflows/cmd/wfcli/swagger-client/models"
)

// WfiValidateReader is a Reader for the WfiValidate structure.
type WfiValidateReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *WfiValidateReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewWfiValidateOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		return nil, runtime.NewAPIError("unknown error", response, response.Code())
	}
}

// NewWfiValidateOK creates a WfiValidateOK with default headers values
func NewWfiValidateOK() *WfiValidateOK {
	return &WfiValidateOK{}
}

/*WfiValidateOK handles this case with default header values.

WfiValidateOK wfi validate o k
*/
type WfiValidateOK struct {
	Payload models.ProtobufEmpty
}

func (o *WfiValidateOK) Error() string {
	return fmt.Sprintf("[POST /invocation/validate][%d] wfiValidateOK  %+v", 200, o.Payload)
}

func (o *WfiValidateOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
