package iam

import (
	iamadmin "cloud.google.com/go/iam"
	admin "cloud.google.com/go/iam/admin/apiv1"
	credentials "cloud.google.com/go/iam/credentials/apiv1"
	"context"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/ktool-backend/database"
	"github.com/YasiruR/ktool-backend/domain"
	"github.com/YasiruR/ktool-backend/log"
	oauth2 "golang.org/x/oauth2/google"
	resource "google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
	adminpb "google.golang.org/genproto/googleapis/iam/admin/v1"
	credentialspb "google.golang.org/genproto/googleapis/iam/credentials/v1"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
)

//func main() {
//	ctx := context.Background()
//	res, err := validateGKESecret("1")
//	if err != nil {
//		log.Logger.ErrorContext(ctx, "Could not retrieve cluster list")
//		return
//	}
//	log.Logger.InfoContext(ctx, "Successfully retrieved cluster list from GKE")
//	fmt.Println(res)
//}

//TODO: use resource-manager api to validate the secret
func TestIamPermissions(userId string) (*resource.TestIamPermissionsResponse, error) {
	ctx := context.Background()
	b, cred, err := GetGkeCredentialsForUser(userId)
	if err != nil {
		return nil, err
	}
	conf, err := oauth2.JWTConfigFromJSON(b, resource.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	resourceService, err := resource.NewService(ctx, option.WithTokenSource(conf.TokenSource(ctx)))
	if err != nil {
		return nil, err
	}
	req := &resource.TestIamPermissionsRequest{
		Permissions: []string{ //todo: read from file
			"container.clusterRoleBindings.get",
			"container.clusterRoleBindings.list",
			"container.clusterRoleBindings.update",
			"container.clusterRoles.bind",
			"container.clusterRoles.create",
			"container.clusterRoles.delete",
			"container.clusterRoles.get",
			"container.clusterRoles.list",
			"container.clusterRoles.update",
			"container.clusters.create",
			"container.clusters.delete",
			"container.clusters.get",
			"container.clusters.getCredentials",
			"container.clusters.list",
			"container.clusters.update",
			"container.componentStatuses.get",
			"container.componentStatuses.list",
			"container.configMaps.create",
			"container.configMaps.delete",
			"container.configMaps.get",
			"container.configMaps.list",
			"container.configMaps.update",
			"container.controllerRevisions.create",
			"container.controllerRevisions.delete",
			"container.controllerRevisions.get",
			"container.controllerRevisions.list",
			"container.controllerRevisions.update",
			"container.cronJobs.create",
			"container.cronJobs.delete",
			"container.cronJobs.get",
			"container.cronJobs.getStatus",
			"container.cronJobs.list",
			"container.cronJobs.update",
			"container.cronJobs.updateStatus",
		},
	}
	resp, err := resourceService.Projects.TestIamPermissions(cred.ProjectId, req).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func GetOAuthAccessToken(userId string) (*credentialspb.GenerateAccessTokenResponse, error) {
	ctx := context.Background()
	b, cred, err := GetGkeCredentialsForUser(userId)
	//log.Logger.Info(cred)
	if err != nil {
		return nil, err
	}
	c, err := credentials.NewIamCredentialsClient(ctx, option.WithCredentialsJSON(b))
	if err != nil {
		// TODO: Handle error.
	}

	req := &credentialspb.GenerateAccessTokenRequest{
		Name: fmt.Sprintf("projects/-/serviceAccounts/%s", cred.ClientId),
		Delegates: []string{
			fmt.Sprintf("projects/-/serviceAccounts/%s", cred.ClientId),
		},
		Scope: []string{
			"https://www.googleapis.com/auth/cloud-platform",
		},
	}
	resp, err := c.GenerateAccessToken(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	return resp, nil
}

func GetServiceAccount(userId string) (*adminpb.ServiceAccount, error) {
	ctx := context.Background()
	b, cred, err := GetGkeCredentialsForUser(userId)
	//log.Logger.Info(cred)
	if err != nil {
		return nil, err
	}
	c, err := admin.NewIamClient(ctx, option.WithCredentialsJSON(b))
	if err != nil {
		// TODO: Handle error.
	}

	req := &adminpb.GetServiceAccountRequest{
		Name: fmt.Sprintf("projects/%s/serviceAccounts/%s", cred.ProjectId, cred.ClientId),
	}
	resp, err := c.GetServiceAccount(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	return resp, nil
}

func ListRolesForServiceAccount(userId string) (*adminpb.ListRolesResponse, error) {
	ctx := context.Background()
	b, cred, err := GetGkeCredentialsForUser(userId)
	//log.Logger.Info(cred)
	if err != nil {
		return nil, err
	}
	c, err := admin.NewIamClient(ctx, option.WithCredentialsJSON(b))
	if err != nil {
		// TODO: Handle error.
	}

	req := &adminpb.ListRolesRequest{
		Parent:   fmt.Sprintf("projects/%s", cred.ProjectId),
		View:     1,
		PageSize: 1000,
	}
	resp, err := c.ListRoles(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	return resp, nil
}

func TestIamPermissionsForServiceAcc(userId string) (**iampb.TestIamPermissionsResponse, error) {
	ctx := context.Background()
	b, cred, err := GetGkeCredentialsForUser(userId)
	//log.Logger.Info(cred)
	if err != nil {
		return nil, err
	}
	c, err := admin.NewIamClient(ctx, option.WithCredentialsJSON(b))
	if err != nil {
		// TODO: Handle error.
	}

	req := &iampb.TestIamPermissionsRequest{
		// TODO: Fill request struct fields.
		Resource: fmt.Sprintf("projects/%s/serviceAccounts/%s", cred.ProjectId, cred.ClientMail),
		Permissions: []string{
			"iam.serviceAccounts.actAs",
			"iam.serviceAccounts.get",
			"iam.serviceAccounts.list",
		},
	}
	resp, err := c.TestIamPermissions(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	return &resp, nil
}

func GetServiceAccountIamPolicies(userId string) (*iamadmin.Policy, error) {
	ctx := context.Background()
	b, cred, err := GetGkeCredentialsForUser(userId)
	//log.Logger.Info(cred)
	if err != nil {
		return nil, err
	}
	c, err := admin.NewIamClient(ctx, option.WithCredentialsJSON(b))
	if err != nil {
		// TODO: Handle error.
	}

	//req := &iamadmin.ListRolesRequest{
	//	// TODO: Fill request struct fields.
	//	Parent: "projects/ktool-280018",
	//	View: 1,
	//}
	req := &iampb.GetIamPolicyRequest{
		// TODO: Fill request struct fields.
		Resource: "projects/" + cred.ProjectId + "/serviceAccounts/" + cred.ClientId,
	}
	resp, err := c.GetIamPolicy(ctx, req)
	if err != nil {
		// TODO: Handle error.
	}
	// TODO: Use resp.
	return resp, nil
}

func GetGkeCredentialsForUser(userId string) ([]byte, domain.GkeSecret, error) {
	ctx := context.Background()
	secretDao := database.GetSecretInternal(ctx, userId, `Google`, `ktool-gke`)

	if err := secretDao.Error; err != nil {
		log.Logger.ErrorContext(ctx, "Error occurred while fetching eks secret for client %s", userId)
		return nil, domain.GkeSecret{}, err
	}
	cred := domain.GkeSecret{
		Type:              secretDao.Secret.GkeType,
		ProjectId:         secretDao.Secret.GkeProjectId,
		PrivateKeyId:      secretDao.Secret.GkePrivateKeyId,
		PrivateKey:        secretDao.Secret.GkePrivateKey,
		ClientMail:        secretDao.Secret.GkeClientMail,
		ClientId:          secretDao.Secret.GkeClientId,
		AuthUri:           secretDao.Secret.GkeAuthUri,
		TokenUri:          secretDao.Secret.GkeTokenUri,
		AuthX509CertUrl:   secretDao.Secret.GkeAuthX509CertUrl,
		ClientX509CertUrl: secretDao.Secret.GkeClientX509CertUrl,
	}
	bytes, err := json.Marshal(&cred)
	if err != nil {
		log.Logger.ErrorContext(ctx, "Could not marshall gke credentials for user %s", userId)
		return nil, cred, err
	}
	return bytes, cred, nil
}
