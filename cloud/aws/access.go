package aws

import (
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
)

type Secu struct {
	*sts.STS
}

func NewSecu(sess *session.Session) *Secu {
	return &Secu{sts.New(sess)}
}

func (s *Secu) CallerIdentity() (interface{}, error) {
	return s.GetCallerIdentity(&sts.GetCallerIdentityInput{})
}

func (s *Secu) GetUserId() (string, error) {
	output, err := s.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return awssdk.StringValue(output.Arn), nil
}

func (s *Secu) GetAccountId() (string, error) {
	output, err := s.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return awssdk.StringValue(output.Account), nil
}

type Access struct {
	*iam.IAM
}

func NewAccess(sess *session.Session) *Access {
	return &Access{IAM: iam.New(sess)}
}

func (a *Access) Users() (interface{}, error) {
	return a.ListUsers(&iam.ListUsersInput{})
}

func (a *Access) Groups() (interface{}, error) {
	return a.ListGroups(&iam.ListGroupsInput{})
}

func (a *Access) Roles() (interface{}, error) {
	return a.ListRoles(&iam.ListRolesInput{})
}

func (a *Access) LocalPolicies() (interface{}, error) {
	return a.ListPolicies(&iam.ListPoliciesInput{Scope: awssdk.String(iam.PolicyScopeTypeLocal)})
}

func (a *Access) AccountDetails() (interface{}, error) {
	params := &iam.GetAccountAuthorizationDetailsInput{
		Filter: []*string{
			awssdk.String(iam.EntityTypeUser),
			awssdk.String(iam.EntityTypeRole),
			awssdk.String(iam.EntityTypeGroup),
			awssdk.String(iam.EntityTypeLocalManagedPolicy),
			awssdk.String(iam.EntityTypeAwsmanagedPolicy),
		},
	}
	return a.GetAccountAuthorizationDetails(params)
}

type AwsAccess struct {
	Users []*iam.User

	GroupsDetail []*iam.GroupDetail
	UsersDetail  []*iam.UserDetail
	RolesDetail  []*iam.RoleDetail
	Policies     []*iam.ManagedPolicyDetail

	UserGroups map[string][]string

	UserPolicies  map[string][]string
	GroupPolicies map[string][]string
	RolePolicies  map[string][]string
}

func NewAwsAccess() *AwsAccess {
	return &AwsAccess{
		UserGroups:    make(map[string][]string),
		UserPolicies:  make(map[string][]string),
		GroupPolicies: make(map[string][]string),
		RolePolicies:  make(map[string][]string),
	}
}

func (access *Access) FetchAwsAccess() (*AwsAccess, error) {
	resultc, errc := multiFetch(access.AccountDetails, access.Users)

	awsAccess := NewAwsAccess()

	for r := range resultc {
		switch rr := r.(type) {
		case *iam.ListUsersOutput:
			awsAccess.Users = append(awsAccess.Users, rr.Users...)

		case *iam.GetAccountAuthorizationDetailsOutput:
			for _, user := range rr.UserDetailList {
				awsAccess.UsersDetail = append(awsAccess.UsersDetail, user)

				groups := []string{}
				for _, groupId := range user.GroupList {
					groups = append(groups, awssdk.StringValue(groupId))
				}
				awsAccess.UserGroups[awssdk.StringValue(user.UserId)] = groups

				policies := []string{}
				for _, policy := range user.UserPolicyList {
					policies = append(policies, awssdk.StringValue(policy.PolicyName))
				}
				for _, policy := range user.AttachedManagedPolicies {
					policies = append(policies, awssdk.StringValue(policy.PolicyName))
				}
				awsAccess.UserPolicies[awssdk.StringValue(user.UserId)] = policies
			}

			for _, group := range rr.GroupDetailList {
				awsAccess.GroupsDetail = append(awsAccess.GroupsDetail, group)

				policies := []string{}
				for _, policy := range group.GroupPolicyList {
					policies = append(policies, awssdk.StringValue(policy.PolicyName))
				}
				for _, policy := range group.AttachedManagedPolicies {
					policies = append(policies, awssdk.StringValue(policy.PolicyName))
				}
				awsAccess.GroupPolicies[awssdk.StringValue(group.GroupId)] = policies
			}

			for _, role := range rr.RoleDetailList {
				awsAccess.RolesDetail = append(awsAccess.RolesDetail, role)

				policies := []string{}
				for _, policy := range role.RolePolicyList {
					policies = append(policies, awssdk.StringValue(policy.PolicyName))
				}
				for _, policy := range role.AttachedManagedPolicies {
					policies = append(policies, awssdk.StringValue(policy.PolicyName))
				}
				awsAccess.RolePolicies[awssdk.StringValue(role.RoleId)] = policies
			}

			for _, policy := range rr.Policies {
				awsAccess.Policies = append(awsAccess.Policies, policy)
			}
		}
	}

	return awsAccess, <-errc
}
