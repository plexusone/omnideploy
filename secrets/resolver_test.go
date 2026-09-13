package secrets

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/plexusone/omnideploy/config"
)

type fakeSSM struct {
	params map[string]string
	err    error
}

func (f *fakeSSM) GetParameter(_ context.Context, in *ssm.GetParameterInput, _ ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	if in.WithDecryption == nil || !*in.WithDecryption {
		return nil, errors.New("fakeSSM: WithDecryption must be set for SecureString parameters")
	}
	v, ok := f.params[*in.Name]
	if !ok {
		return nil, errors.New("ParameterNotFound")
	}
	return &ssm.GetParameterOutput{Parameter: &ssmtypes.Parameter{Value: aws.String(v)}}, nil
}

type fakeSecretsManager struct {
	secrets map[string]string
}

func (f *fakeSecretsManager) GetSecretValue(_ context.Context, in *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	v, ok := f.secrets[*in.SecretId]
	if !ok {
		return nil, errors.New("ResourceNotFoundException")
	}
	return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(v)}, nil
}

func newTestResolver() *Resolver {
	return &Resolver{
		SSM:            &fakeSSM{params: map[string]string{"/myapp/api-key": "ssm-value"}},
		SecretsManager: &fakeSecretsManager{secrets: map[string]string{"myapp/token": "sm-value"}},
		Getenv: func(k string) string {
			if k == "MY_ENV_SECRET" {
				return "env-value"
			}
			return ""
		},
	}
}

func TestResolve_AllSources(t *testing.T) {
	r := newTestResolver()
	got, err := r.Resolve(context.Background(), []config.SecretRef{
		{Name: "FROM_ENV", Source: "env:MY_ENV_SECRET"},
		{Name: "FROM_SSM", Source: "ssm:/myapp/api-key"},
		{Name: "FROM_SM", Source: "secretsmanager:myapp/token"},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := map[string]string{"FROM_ENV": "env-value", "FROM_SSM": "ssm-value", "FROM_SM": "sm-value"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("Resolve()[%s] = %q, want %q", k, got[k], v)
		}
	}
}

func TestResolve_Failures(t *testing.T) {
	tests := []struct {
		name    string
		ref     config.SecretRef
		wantErr string
	}{
		{"unset env var", config.SecretRef{Name: "X", Source: "env:NOPE"}, "unset or empty"},
		{"missing ssm param", config.SecretRef{Name: "X", Source: "ssm:/absent"}, "ParameterNotFound"},
		{"missing sm secret", config.SecretRef{Name: "X", Source: "secretsmanager:absent"}, "ResourceNotFoundException"},
		{"unsupported source", config.SecretRef{Name: "X", Source: "vault:kv/x"}, "unsupported source"},
		{"empty name", config.SecretRef{Name: "", Source: "env:A"}, "must set both name and source"},
		{"empty source", config.SecretRef{Name: "X", Source: ""}, "must set both name and source"},
	}
	r := newTestResolver()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.Resolve(context.Background(), []config.SecretRef{tt.ref})
			if err == nil {
				t.Fatal("Resolve() succeeded, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Resolve() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestResolve_NilAWSClients(t *testing.T) {
	r := &Resolver{Getenv: func(string) string { return "v" }}
	if _, err := r.Resolve(context.Background(), []config.SecretRef{{Name: "X", Source: "ssm:/p"}}); err == nil || !strings.Contains(err.Error(), "no SSM client") {
		t.Errorf("ssm without client: err = %v, want no-SSM-client error", err)
	}
	if _, err := r.Resolve(context.Background(), []config.SecretRef{{Name: "X", Source: "secretsmanager:s"}}); err == nil || !strings.Contains(err.Error(), "no Secrets Manager client") {
		t.Errorf("secretsmanager without client: err = %v, want no-client error", err)
	}
	// env: works with no AWS clients at all.
	got, err := r.Resolve(context.Background(), []config.SecretRef{{Name: "X", Source: "env:ANY"}})
	if err != nil || got["X"] != "v" {
		t.Errorf("env-only resolve = (%v, %v), want (map[X:v], nil)", got, err)
	}
}

func TestResolve_EmptyRefs(t *testing.T) {
	r := &Resolver{}
	got, err := r.Resolve(context.Background(), nil)
	if err != nil || len(got) != 0 {
		t.Errorf("Resolve(nil) = (%v, %v), want (empty, nil)", got, err)
	}
}
