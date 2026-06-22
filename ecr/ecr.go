// Package ecr provides ECR repository management.
package ecr

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

// Client wraps the ECR service client.
type Client struct {
	client *ecr.Client
	region string
}

// New creates a new ECR client.
func New(ctx context.Context, region string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Client{
		client: ecr.NewFromConfig(cfg),
		region: region,
	}, nil
}

// Repository represents an ECR repository.
type Repository struct {
	Name string
	URI  string
	ARN  string
}

// CreateRepository creates a new ECR repository.
func (c *Client) CreateRepository(ctx context.Context, name string) (*Repository, error) {
	out, err := c.client.CreateRepository(ctx, &ecr.CreateRepositoryInput{
		RepositoryName:     aws.String(name),
		ImageTagMutability: types.ImageTagMutabilityMutable,
		ImageScanningConfiguration: &types.ImageScanningConfiguration{
			ScanOnPush: true,
		},
	})
	if err != nil {
		return nil, err
	}

	return &Repository{
		Name: aws.ToString(out.Repository.RepositoryName),
		URI:  aws.ToString(out.Repository.RepositoryUri),
		ARN:  aws.ToString(out.Repository.RepositoryArn),
	}, nil
}

// GetRepository gets an existing ECR repository.
func (c *Client) GetRepository(ctx context.Context, name string) (*Repository, error) {
	out, err := c.client.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{name},
	})
	if err != nil {
		return nil, err
	}

	if len(out.Repositories) == 0 {
		return nil, fmt.Errorf("repository not found: %s", name)
	}

	repo := out.Repositories[0]
	return &Repository{
		Name: aws.ToString(repo.RepositoryName),
		URI:  aws.ToString(repo.RepositoryUri),
		ARN:  aws.ToString(repo.RepositoryArn),
	}, nil
}

// EnsureRepository creates a repository if it doesn't exist.
func (c *Client) EnsureRepository(ctx context.Context, name string) (*Repository, bool, error) {
	// Try to get existing
	repo, err := c.GetRepository(ctx, name)
	if err == nil {
		return repo, false, nil // Already exists
	}

	// Create new
	repo, err = c.CreateRepository(ctx, name)
	if err != nil {
		return nil, false, err
	}

	return repo, true, nil
}

// ListRepositories lists all ECR repositories.
func (c *Client) ListRepositories(ctx context.Context) ([]Repository, error) {
	var repos []Repository

	paginator := ecr.NewDescribeRepositoriesPaginator(c.client, &ecr.DescribeRepositoriesInput{})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, r := range out.Repositories {
			repos = append(repos, Repository{
				Name: aws.ToString(r.RepositoryName),
				URI:  aws.ToString(r.RepositoryUri),
				ARN:  aws.ToString(r.RepositoryArn),
			})
		}
	}

	return repos, nil
}

// DeleteRepository deletes an ECR repository.
func (c *Client) DeleteRepository(ctx context.Context, name string, force bool) error {
	_, err := c.client.DeleteRepository(ctx, &ecr.DeleteRepositoryInput{
		RepositoryName: aws.String(name),
		Force:          force, // Delete even if images exist
	})
	return err
}

// LoginCredentials holds Docker login credentials.
type LoginCredentials struct {
	Username string
	Password string
	Registry string
	Command  string
}

// GetLoginCredentials gets Docker login credentials for ECR.
func (c *Client) GetLoginCredentials(ctx context.Context) (*LoginCredentials, error) {
	out, err := c.client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return nil, err
	}

	if len(out.AuthorizationData) == 0 {
		return nil, fmt.Errorf("no authorization data returned")
	}

	auth := out.AuthorizationData[0]
	token, err := base64.StdEncoding.DecodeString(aws.ToString(auth.AuthorizationToken))
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	// Token is "AWS:<password>"
	parts := strings.SplitN(string(token), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid token format")
	}

	registry := aws.ToString(auth.ProxyEndpoint)
	registry = strings.TrimPrefix(registry, "https://")

	return &LoginCredentials{
		Username: parts[0],
		Password: parts[1],
		Registry: registry,
		Command:  fmt.Sprintf("docker login --username AWS --password-stdin %s", registry),
	}, nil
}
