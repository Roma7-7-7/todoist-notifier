// Package ssm provides utilities for fetching AWS Systems Manager (SSM) Parameter Store values.
//
// This package simplifies retrieving multiple SSM parameters with automatic decryption support
// and validation. It's designed to be reusable across different services and applications.
//
// Example usage:
//
//	cfg, _ := config.LoadDefaultConfig(ctx)
//	client := ssm.NewFromConfig(cfg)
//
//	var apiKey, dbPassword string
//	err := ssm.FetchParameters(ctx, client, map[string]*string{
//		"/app/prod/api-key":     &apiKey,
//		"/app/prod/db-password": &dbPassword,
//	}, ssm.WithDecryption())
package ssm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// ErrParamsNotFound is returned when one or more requested parameters
// do not exist in AWS Systems Manager Parameter Store.
var ErrParamsNotFound = errors.New("params not found")

// Client defines the interface for SSM operations.
// This abstraction allows for easier testing and decoupling from the AWS SDK.
type Client interface {
	GetParameters(ctx context.Context, params *ssm.GetParametersInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersOutput, error)
}

// FetchOptions configures the behavior of parameter fetching.
type FetchOptions struct {
	// withDecryption enables automatic decryption of SecureString parameters
	withDecryption bool
}

// OptionsF is a functional option for configuring FetchParameters.
type OptionsF func(*FetchOptions)

// WithDecryption returns an option that enables automatic decryption
// of SecureString parameters when fetching from SSM.
func WithDecryption() OptionsF {
	return func(o *FetchOptions) {
		o.withDecryption = true
	}
}

// FetchParameters retrieves multiple SSM parameters and populates their destination pointers.
//
// Parameters:
//   - ctx: Context for the request
//   - client: SSM client implementation
//   - params: Map of parameter names to destination string pointers
//   - opts: Optional functional options (e.g., WithDecryption())
//
// Returns an error if:
//   - The SSM API call fails
//   - Any requested parameter does not exist in SSM (returns ErrParamsNotFound)
//
// Example:
//
//	var token, chatID string
//	err := FetchParameters(ctx, ssmClient, map[string]*string{
//		"/app/prod/token":   &token,
//		"/app/prod/chat-id": &chatID,
//	}, WithDecryption())
func FetchParameters(ctx context.Context, client Client, params map[string]*string, opts ...OptionsF) error {
	if len(params) == 0 {
		return nil
	}

	options := &FetchOptions{}
	for _, o := range opts {
		o(options)
	}

	names := make([]string, len(params))

	i := 0
	for name := range params {
		names[i] = name
		i++
	}

	result, err := client.GetParameters(ctx, &ssm.GetParametersInput{
		Names:          names,
		WithDecryption: aws.Bool(options.withDecryption),
	})
	if err != nil {
		return fmt.Errorf("ssm get parameters: %w", err)
	}

	if len(result.InvalidParameters) > 0 {
		return fmt.Errorf("%w: %s", ErrParamsNotFound, strings.Join(result.InvalidParameters, ", "))
	}

	for _, param := range result.Parameters {
		if param.Name == nil || param.Value == nil {
			continue
		}
		if dest, ok := params[*param.Name]; ok {
			*dest = *param.Value
		}
	}

	return nil
}
