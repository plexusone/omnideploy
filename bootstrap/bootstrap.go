// Package bootstrap provides IAM setup for OmniDeploy.
package bootstrap

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

//go:embed policy.json
var policyDocument string

// Config holds bootstrap configuration.
type Config struct {
	// PolicyName is the name of the IAM policy to create.
	PolicyName string

	// GroupName is the name of the IAM group to create.
	GroupName string

	// UserName is the name of the IAM user to create (optional).
	UserName string

	// CreateAccessKey creates an access key for the user.
	CreateAccessKey bool

	// Region is the AWS region.
	Region string
}

// DefaultConfig returns the default bootstrap configuration.
func DefaultConfig() Config {
	return Config{
		PolicyName: "OmniDeployPolicy",
		GroupName:  "omnideploy-users",
		Region:     "us-west-2", // Oregon - LightSail containers supported
	}
}

// Result holds the bootstrap results.
type Result struct {
	PolicyARN       string
	GroupName       string
	UserName        string
	AccessKeyID     string
	SecretAccessKey string
}

// Bootstrapper sets up IAM resources for OmniDeploy.
type Bootstrapper struct {
	client *iam.Client
	cfg    Config
}

// New creates a new Bootstrapper.
func New(ctx context.Context, cfg Config) (*Bootstrapper, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Bootstrapper{
		client: iam.NewFromConfig(awsCfg),
		cfg:    cfg,
	}, nil
}

// Run executes the bootstrap process.
func (b *Bootstrapper) Run(ctx context.Context) (*Result, error) {
	result := &Result{}

	// 1. Create or get policy
	policyARN, err := b.ensurePolicy(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}
	result.PolicyARN = policyARN

	// 2. Create or get group
	groupName, err := b.ensureGroup(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}
	result.GroupName = groupName

	// 3. Attach policy to group
	if err := b.attachPolicyToGroup(ctx, policyARN, groupName); err != nil {
		return nil, fmt.Errorf("failed to attach policy to group: %w", err)
	}

	// 4. Create user if requested
	if b.cfg.UserName != "" {
		userName, err := b.ensureUser(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
		result.UserName = userName

		// Add user to group
		if err := b.addUserToGroup(ctx, userName, groupName); err != nil {
			return nil, fmt.Errorf("failed to add user to group: %w", err)
		}

		// Create access key if requested
		if b.cfg.CreateAccessKey {
			keyID, secret, err := b.createAccessKey(ctx, userName)
			if err != nil {
				return nil, fmt.Errorf("failed to create access key: %w", err)
			}
			result.AccessKeyID = keyID
			result.SecretAccessKey = secret
		}
	}

	return result, nil
}

func (b *Bootstrapper) ensurePolicy(ctx context.Context) (string, error) {
	// Check if policy exists
	listOut, err := b.client.ListPolicies(ctx, &iam.ListPoliciesInput{
		Scope: types.PolicyScopeTypeLocal,
	})
	if err != nil {
		return "", err
	}

	for _, p := range listOut.Policies {
		if aws.ToString(p.PolicyName) == b.cfg.PolicyName {
			return aws.ToString(p.Arn), nil
		}
	}

	// Create policy
	createOut, err := b.client.CreatePolicy(ctx, &iam.CreatePolicyInput{
		PolicyName:     aws.String(b.cfg.PolicyName),
		PolicyDocument: aws.String(policyDocument),
		Description:    aws.String("OmniDeploy deployment permissions for LightSail, ECR, and SSM"),
	})
	if err != nil {
		return "", err
	}

	return aws.ToString(createOut.Policy.Arn), nil
}

func (b *Bootstrapper) ensureGroup(ctx context.Context) (string, error) {
	// Check if group exists
	_, err := b.client.GetGroup(ctx, &iam.GetGroupInput{
		GroupName: aws.String(b.cfg.GroupName),
	})
	if err == nil {
		return b.cfg.GroupName, nil
	}

	var notFound *types.NoSuchEntityException
	if !errors.As(err, &notFound) {
		return "", err
	}

	// Create group
	_, err = b.client.CreateGroup(ctx, &iam.CreateGroupInput{
		GroupName: aws.String(b.cfg.GroupName),
	})
	if err != nil {
		return "", err
	}

	return b.cfg.GroupName, nil
}

func (b *Bootstrapper) attachPolicyToGroup(ctx context.Context, policyARN, groupName string) error {
	// Check if already attached
	listOut, err := b.client.ListAttachedGroupPolicies(ctx, &iam.ListAttachedGroupPoliciesInput{
		GroupName: aws.String(groupName),
	})
	if err != nil {
		return err
	}

	for _, p := range listOut.AttachedPolicies {
		if aws.ToString(p.PolicyArn) == policyARN {
			return nil // Already attached
		}
	}

	// Attach policy
	_, err = b.client.AttachGroupPolicy(ctx, &iam.AttachGroupPolicyInput{
		GroupName: aws.String(groupName),
		PolicyArn: aws.String(policyARN),
	})
	return err
}

func (b *Bootstrapper) ensureUser(ctx context.Context) (string, error) {
	// Check if user exists
	_, err := b.client.GetUser(ctx, &iam.GetUserInput{
		UserName: aws.String(b.cfg.UserName),
	})
	if err == nil {
		return b.cfg.UserName, nil
	}

	var notFound *types.NoSuchEntityException
	if !errors.As(err, &notFound) {
		return "", err
	}

	// Create user
	_, err = b.client.CreateUser(ctx, &iam.CreateUserInput{
		UserName: aws.String(b.cfg.UserName),
	})
	if err != nil {
		return "", err
	}

	return b.cfg.UserName, nil
}

func (b *Bootstrapper) addUserToGroup(ctx context.Context, userName, groupName string) error {
	_, err := b.client.AddUserToGroup(ctx, &iam.AddUserToGroupInput{
		UserName:  aws.String(userName),
		GroupName: aws.String(groupName),
	})

	// Ignore if already in group
	var entityExists *types.EntityAlreadyExistsException
	if errors.As(err, &entityExists) {
		return nil
	}

	return err
}

func (b *Bootstrapper) createAccessKey(ctx context.Context, userName string) (string, string, error) {
	out, err := b.client.CreateAccessKey(ctx, &iam.CreateAccessKeyInput{
		UserName: aws.String(userName),
	})
	if err != nil {
		return "", "", err
	}

	return aws.ToString(out.AccessKey.AccessKeyId), aws.ToString(out.AccessKey.SecretAccessKey), nil
}

// Status checks the current bootstrap status.
func (b *Bootstrapper) Status(ctx context.Context) (*StatusResult, error) {
	result := &StatusResult{}

	// Check policy
	listOut, err := b.client.ListPolicies(ctx, &iam.ListPoliciesInput{
		Scope: types.PolicyScopeTypeLocal,
	})
	if err != nil {
		return nil, err
	}
	for _, p := range listOut.Policies {
		if aws.ToString(p.PolicyName) == b.cfg.PolicyName {
			result.PolicyExists = true
			result.PolicyARN = aws.ToString(p.Arn)
			break
		}
	}

	// Check group
	groupOut, err := b.client.GetGroup(ctx, &iam.GetGroupInput{
		GroupName: aws.String(b.cfg.GroupName),
	})
	if err == nil {
		result.GroupExists = true
		result.GroupName = b.cfg.GroupName
		for _, u := range groupOut.Users {
			result.GroupMembers = append(result.GroupMembers, aws.ToString(u.UserName))
		}
	}

	return result, nil
}

// StatusResult holds the bootstrap status.
type StatusResult struct {
	PolicyExists bool
	PolicyARN    string
	GroupExists  bool
	GroupName    string
	GroupMembers []string
}

// PolicyJSON returns the policy document as formatted JSON.
func PolicyJSON() string {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(policyDocument), &obj); err != nil {
		return policyDocument
	}
	formatted, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return policyDocument
	}
	return string(formatted)
}
