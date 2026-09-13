// Package secrets resolves config.SecretRef sources to values at deploy
// time (RMI-OMNIAGENT-006). Resolution happens in the deploying process,
// before the Pulumi program runs; backends are expected to mark resolved
// values as Pulumi secrets so they are encrypted in state. Note the
// Lightsail ceiling: containers have no task IAM role or native secret
// references, so resolved values still become container environment
// variables visible in the Lightsail console — deploy-time resolution
// centralizes storage, rotation, and audit, not console visibility.
package secrets

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/plexusone/omnideploy/config"
)

// SSMClient is the subset of the SSM API the resolver uses.
type SSMClient interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

// SecretsManagerClient is the subset of the Secrets Manager API the
// resolver uses.
type SecretsManagerClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// Resolver resolves SecretRef sources. Zero-value fields fall back to
// real implementations where possible (Getenv → os.Getenv); AWS-backed
// sources require the corresponding client and error otherwise.
type Resolver struct {
	SSM            SSMClient
	SecretsManager SecretsManagerClient
	Getenv         func(string) string
}

// NewResolver builds a Resolver with real AWS clients for the region,
// using the default credential chain (env, shared config, SSO, IMDS).
func NewResolver(ctx context.Context, region string) (*Resolver, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	return &Resolver{
		SSM:            ssm.NewFromConfig(awsCfg),
		SecretsManager: secretsmanager.NewFromConfig(awsCfg),
		Getenv:         os.Getenv,
	}, nil
}

// Resolve maps each SecretRef's Name to its resolved value. Supported
// sources:
//
//	env:VAR             — the deploying shell's environment
//	ssm:/path           — SSM Parameter Store (SecureString decrypted)
//	secretsmanager:name — AWS Secrets Manager (SecretString)
//
// Every ref must resolve to a non-empty value; a missing secret fails
// the deploy up front rather than shipping a container with a silently
// absent credential.
func (r *Resolver) Resolve(ctx context.Context, refs []config.SecretRef) (map[string]string, error) {
	getenv := r.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}

	out := make(map[string]string, len(refs))
	for _, ref := range refs {
		if ref.Name == "" || ref.Source == "" {
			return nil, fmt.Errorf("secret ref must set both name and source (name=%q, source=%q)", ref.Name, ref.Source)
		}

		switch {
		case strings.HasPrefix(ref.Source, "env:"):
			varName := strings.TrimPrefix(ref.Source, "env:")
			v := getenv(varName)
			if v == "" {
				return nil, fmt.Errorf("secret %s: environment variable %s is unset or empty", ref.Name, varName)
			}
			out[ref.Name] = v

		case strings.HasPrefix(ref.Source, "ssm:"):
			if r.SSM == nil {
				return nil, fmt.Errorf("secret %s: no SSM client configured", ref.Name)
			}
			paramName := strings.TrimPrefix(ref.Source, "ssm:")
			resp, err := r.SSM.GetParameter(ctx, &ssm.GetParameterInput{
				Name:           aws.String(paramName),
				WithDecryption: aws.Bool(true),
			})
			if err != nil {
				return nil, fmt.Errorf("secret %s: ssm get-parameter %s: %w", ref.Name, paramName, err)
			}
			if resp.Parameter == nil || resp.Parameter.Value == nil || *resp.Parameter.Value == "" {
				return nil, fmt.Errorf("secret %s: ssm parameter %s has no value", ref.Name, paramName)
			}
			out[ref.Name] = *resp.Parameter.Value

		case strings.HasPrefix(ref.Source, "secretsmanager:"):
			if r.SecretsManager == nil {
				return nil, fmt.Errorf("secret %s: no Secrets Manager client configured", ref.Name)
			}
			secretID := strings.TrimPrefix(ref.Source, "secretsmanager:")
			resp, err := r.SecretsManager.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
				SecretId: aws.String(secretID),
			})
			if err != nil {
				return nil, fmt.Errorf("secret %s: secretsmanager get-secret-value %s: %w", ref.Name, secretID, err)
			}
			if resp.SecretString == nil || *resp.SecretString == "" {
				return nil, fmt.Errorf("secret %s: secretsmanager secret %s has no string value", ref.Name, secretID)
			}
			out[ref.Name] = *resp.SecretString

		default:
			return nil, fmt.Errorf("secret %s: unsupported source %q (want env:, ssm:, or secretsmanager:)", ref.Name, ref.Source)
		}
	}
	return out, nil
}
