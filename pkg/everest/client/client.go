// Package client provides helpers to communicate with Everest API
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/percona/percona-everest-backend/client"
	"github.com/pkg/errors"
)

// Everest is a connector to the Everest API.
type Everest struct {
	cl *client.Client
}

// NewEverest returns new Everest.
func NewEverest(everestClient *client.Client) *Everest {
	return &Everest{
		cl: everestClient,
	}
}

// NewEverestFromURL returns a new Everest from a provided URL.
func NewEverestFromURL(url string) (*Everest, error) {
	everestCl, err := client.NewClient(fmt.Sprintf("%s/v1", url))
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize everest client")
	}
	return NewEverest(everestCl), nil
}

// makeRequest calls arbitrary *client.Client method for API call and applies common logic for response handling.
// See methods in Everest struct for examples how to call.
func makeRequest[B interface{}, R interface{}](
	ctx context.Context,
	fn func(context.Context, B, ...client.RequestEditorFn) (*http.Response, error),
	body B,
	ret R,
	errorStatus error,
) error {
	res, err := fn(ctx, body)
	if err != nil {
		return err
	}
	defer res.Body.Close() //nolint:errcheck

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return processErrorResponse(res, errorStatus)
	}

	return json.NewDecoder(res.Body).Decode(ret)
}

func processErrorResponse(res *http.Response, err error) error {
	errMsg := client.Error{}
	if err := json.NewDecoder(res.Body).Decode(&errMsg); err != nil {
		return errors.Wrapf(err, "could not decode Everest error response (status %d)", res.StatusCode)
	}

	msg := fmt.Sprintf("unknown error (status %d)", res.StatusCode)
	if errMsg.Message != nil {
		msg = fmt.Sprintf("%s (status %d)", *errMsg.Message, res.StatusCode)
	}

	return errors.Wrap(err, msg)
}
