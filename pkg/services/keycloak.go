package services

import (
	"context"
	"crypto/tls"

	"github.com/Nerzal/gocloak/v7"
	"github.com/google/uuid"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

const (
	rhOrgId = "rh-org-id"
)

//go:generate moq -out keycloakservice_moq.go . KeycloakService
type KeycloakService interface {
	createClient(client gocloak.Client, accessToken string) (string, *errors.ServiceError)
	getToken() (string, error)
	getClientSecret(internalClientID string, accessToken string) (string, *errors.ServiceError)
	clientConfig(client ClientRepresentation) gocloak.Client
	deleteClient(internalClientId string, accessToken string) *errors.ServiceError
	RegisterKafkaClientInSSO(kafkaNamespace string, orgId string) (string, *errors.ServiceError)
	DeRegisterKafkaClientInSSO(kafkaNamespace string) *errors.ServiceError
	GetSecretForRegisteredKafkaClient(kafkaClusterName string) (string, *errors.ServiceError)
	getClient(clientId string, accessToken string) ([]*gocloak.Client, *errors.ServiceError)
	GetConfig() *config.KeycloakConfig
	IsKafkaClientExist(clientId string) *errors.ServiceError
	CreateServiceAccount(serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError)
	DeleteServiceAccount(ctx context.Context, clientId string) *errors.ServiceError
	ResetServiceAccountCredentials(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError)
	ListServiceAcc(ctx context.Context) ([]api.ServiceAccount, *errors.ServiceError)
}

type keycloakService struct {
	kcClient gocloak.GoCloak
	ctx      context.Context
	config   *config.KeycloakConfig
}

type ClientRepresentation struct {
	Name                         string
	ClientID                     string
	ServiceAccountsEnabled       bool
	Secret                       string
	StandardFlowEnabled          bool
	Attributes                   map[string]string
	AuthorizationServicesEnabled bool
	ProtocolMappers              []gocloak.ProtocolMapperRepresentation
	Description                  string
}

var _ KeycloakService = &keycloakService{}

func NewKeycloakService(config *config.KeycloakConfig) *keycloakService {
	setTokenEndpoints(config)
	client := gocloak.NewClient(config.BaseURL)
	client.RestyClient().SetDebug(config.Debug)
	client.RestyClient().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: config.InsecureSkipVerify})
	return &keycloakService{
		kcClient: client,
		ctx:      context.Background(),
		config:   config,
	}
}

func setTokenEndpoints(config *config.KeycloakConfig) {
	config.JwksEndpointURI = config.BaseURL + "/auth/realms/" + config.Realm + "/protocol/openid-connect/certs"
	config.TokenEndpointURI = config.BaseURL + "/auth/realms/" + config.Realm + "/protocol/openid-connect/token"
	config.ValidIssuerURI = config.BaseURL + "/auth/realms/" + config.Realm
}

//use a token from a Service Account with only the create-client role
func (kc *keycloakService) createClient(client gocloak.Client, accessToken string) (string, *errors.ServiceError) {
	internalClientID, err := kc.kcClient.CreateClient(kc.ctx, accessToken, kc.config.Realm, client)
	if err != nil {
		return "", errors.GeneralError("Failed to create a client: %v", err)
	}
	return internalClientID, nil
}

//internal client name is required to fetch the secrets
func (kc *keycloakService) getClientSecret(internalClientId string, accessToken string) (string, *errors.ServiceError) {
	resp, err := kc.kcClient.GetClientSecret(kc.ctx, accessToken, kc.config.Realm, internalClientId)
	if err != nil {
		return "", errors.GeneralError("Failed to retrieve client secret:%v", err)
	}
	value := *resp.Value
	return value, nil
}

func (kc *keycloakService) getToken() (string, error) {
	options := gocloak.TokenOptions{
		ClientID:     &kc.config.ClientID,
		GrantType:    &kc.config.GrantType,
		ClientSecret: &kc.config.ClientSecret,
	}
	tokenResp, err := kc.kcClient.GetToken(kc.ctx, kc.config.Realm, options)
	if err != nil {
		return "", errors.GeneralError("failed to retrieve the token:%v", err)
	}
	return tokenResp.AccessToken, nil
}

func (kc *keycloakService) RegisterKafkaClientInSSO(kafkaClusterName string, orgId string) (string, *errors.ServiceError) {
	accessToken, _ := kc.getToken()
	internalClientId, err := kc.isClientExist(kafkaClusterName, accessToken)
	if err != nil {
		return "", errors.GeneralError("failed to check the sso client exists:%v", err)
	}
	if internalClientId != "" {
		secretValue, _ := kc.getClientSecret(internalClientId, accessToken)
		return secretValue, nil
	}

	rhOrgIdAttributes := map[string]string{
		rhOrgId: orgId,
	}

	c := ClientRepresentation{
		ClientID:                     kafkaClusterName,
		Name:                         kafkaClusterName,
		ServiceAccountsEnabled:       true,
		AuthorizationServicesEnabled: true,
		StandardFlowEnabled:          false,
		Attributes:                   rhOrgIdAttributes,
	}
	clientConfig := kc.clientConfig(c)
	internalClient, err := kc.createClient(clientConfig, accessToken)
	if err != nil {
		return "", errors.GeneralError("failed to create the sso client:%v", err)
	}
	secretValue, err := kc.getClientSecret(internalClient, accessToken)
	if err != nil {
		return "", errors.GeneralError("failed to get the client secret:%v", err)
	}
	return secretValue, err
}

func (kc *keycloakService) GetSecretForRegisteredKafkaClient(kafkaClusterName string) (string, *errors.ServiceError) {
	accessToken, _ := kc.getToken()
	internalClientId, err := kc.isClientExist(kafkaClusterName, accessToken)
	if err != nil {
		return "", errors.GeneralError("failed to get sso client: %v", err)
	}
	if internalClientId != "" {
		secretValue, _ := kc.getClientSecret(internalClientId, accessToken)
		return secretValue, nil
	}
	return "", nil
}

func (kc *keycloakService) DeRegisterKafkaClientInSSO(kafkaClusterName string) *errors.ServiceError {
	accessToken, _ := kc.getToken()
	internalClientID, _ := kc.isClientExist(kafkaClusterName, accessToken)
	if internalClientID == "" {
		return nil
	}
	err := kc.deleteClient(internalClientID, accessToken)
	if err != nil {
		return errors.GeneralError("failed to delete the sso client:%v", err)
	}
	return nil
}

func (kc *keycloakService) clientConfig(client ClientRepresentation) gocloak.Client {
	return gocloak.Client{
		Name:                         &client.Name,
		ClientID:                     &client.ClientID,
		ServiceAccountsEnabled:       &client.ServiceAccountsEnabled,
		StandardFlowEnabled:          &client.StandardFlowEnabled,
		Attributes:                   &client.Attributes,
		AuthorizationServicesEnabled: &client.AuthorizationServicesEnabled,
		ProtocolMappers:              &client.ProtocolMappers,
		Description:                  &client.Description,
	}
}

func (kc *keycloakService) deleteClient(internalClientID string, accessToken string) *errors.ServiceError {
	err := kc.kcClient.DeleteClient(kc.ctx, accessToken, kc.config.Realm, internalClientID)
	if err != nil {
		return errors.GeneralError("Failed to delete client:%v", err)
	}
	return nil
}

func (kc *keycloakService) getClient(clientId string, accessToken string) ([]*gocloak.Client, *errors.ServiceError) {
	params := gocloak.GetClientsParams{
		ClientID: &clientId,
	}
	client, err := kc.kcClient.GetClients(kc.ctx, accessToken, kc.config.Realm, params)
	if err != nil {
		return nil, errors.GeneralError("Failed to get client:%v", err)
	}
	return client, nil
}

func (kc *keycloakService) getClientById(id string, accessToken string) (*gocloak.Client, *errors.ServiceError) {
	client, err := kc.kcClient.GetClient(kc.ctx, accessToken, kc.config.Realm, id)
	if err != nil {
		return nil, errors.GeneralError("Failed to get client with id %s:%v", id, err)
	}
	return client, nil
}

func (kc *keycloakService) GetConfig() *config.KeycloakConfig {
	return kc.config
}

func (kc *keycloakService) isClientExist(clientId string, accessToken string) (string, *errors.ServiceError) {
	client, err := kc.getClient(clientId, accessToken)
	var internalClientID string
	if err != nil {
		return internalClientID, errors.GeneralError("failed to get sso client: %v", err)
	}
	if len(client) > 0 {
		internalClientID = *client[0].ID
		return internalClientID, nil
	}
	return internalClientID, err
}

func (kc keycloakService) IsKafkaClientExist(clientId string) *errors.ServiceError {
	accessToken, _ := kc.getToken()
	_, err := kc.isClientExist(clientId, accessToken)
	if err != nil {
		return errors.GeneralError("failed to get sso client: %v", err)
	}
	return nil
}
func (kc *keycloakService) CreateServiceAccount(serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
	var serviceAcc api.ServiceAccount
	accessToken, _ := kc.getToken()
	orgId := auth.GetOrgIdFromContext(ctx)
	rhAccountID := map[string][]string{
		"rh-ord-id": {orgId},
	}
	rhOrgIdAttributes := map[string]string{
		rhOrgId: orgId,
	}
	protocolMapper := kc.createProtocolMapperConfig()

	c := ClientRepresentation{
		ClientID:               "srvc-acct-" + NewUUID(),
		Name:                   serviceAccountRequest.Name,
		Description:            serviceAccountRequest.Description,
		ServiceAccountsEnabled: true,
		StandardFlowEnabled:    false,
		ProtocolMappers:        protocolMapper,
		Attributes:             rhOrgIdAttributes,
	}
	clientConfig := kc.clientConfig(c)
	internalClient, err := kc.createClient(clientConfig, accessToken)
	if err != nil {
		return nil, errors.GeneralError("failed to create the service account:%v", err)
	}
	clientSecret, err := kc.getClientSecret(internalClient, accessToken)
	if err != nil {
		return nil, errors.GeneralError("failed to retrieve client secret:%v", err)
	}
	serviceAccountUser, error := kc.kcClient.GetClientServiceAccount(kc.ctx, accessToken, kc.config.Realm, internalClient)
	if error != nil {
		return nil, errors.GeneralError("failed fetch the service account user:%v", err)
	}
	serviceAccountUser.Attributes = &rhAccountID
	error = kc.kcClient.UpdateUser(kc.ctx, accessToken, kc.config.Realm, *serviceAccountUser)
	if error != nil {
		return nil, errors.GeneralError("failed add attributes to service account user:%v", err)
	}
	serviceAcc.ID = internalClient
	serviceAcc.Name = c.Name
	serviceAcc.ClientID = c.ClientID
	serviceAcc.Description = c.Description
	serviceAcc.ClientSecret = clientSecret
	return &serviceAcc, nil
}

func (kc *keycloakService) ListServiceAcc(ctx context.Context) ([]api.ServiceAccount, *errors.ServiceError) {
	accessToken, _ := kc.getToken()
	orgId := auth.GetOrgIdFromContext(ctx)
	params := gocloak.GetClientsParams{}
	var sa []api.ServiceAccount
	clients, err := kc.kcClient.GetClients(kc.ctx, accessToken, kc.config.Realm, params)
	if err != nil {
		return nil, errors.GeneralError("failed to check the sso client exists:%v", err)
	}
	for _, client := range clients {
		acc := api.ServiceAccount{}
		attributes := client.Attributes
		att := *attributes
		if att["rh-org-id"] == orgId {
			acc.ID = *client.ID
			acc.ClientID = *client.ClientID
			acc.Name = safeString(client.Name)
			acc.Description = safeString(client.Description)
			sa = append(sa, acc)
		}
	}
	return sa, nil
}

func (kc *keycloakService) createProtocolMapperConfig() []gocloak.ProtocolMapperRepresentation {
	name := "rh-org-id"
	proto := "openid-connect"
	mapper := "oidc-usermodel-attribute-mapper"
	protocolMapper := []gocloak.ProtocolMapperRepresentation{
		{
			Name:           &name,
			Protocol:       &proto,
			ProtocolMapper: &mapper,
			Config: &map[string]string{
				"access.token.claim":   "true",
				"claim.name":           name,
				"id.token.claim":       "true",
				"jsonType.label":       "String",
				"user.attribute":       name,
				"userinfo.token.claim": "true",
			},
		},
	}
	return protocolMapper
}

func (kc *keycloakService) DeleteServiceAccount(ctx context.Context, id string) *errors.ServiceError {
	accessToken, _ := kc.getToken()
	orgId := auth.GetOrgIdFromContext(ctx)
	c, err := kc.getClientById(id, accessToken)
	if err != nil {
		return errors.GeneralError("failed to check the sso client exists:%v", err)
	}
	if isSameOrg(c, orgId) {
		return kc.deleteClient(id, accessToken)
	} else {
		return errors.Forbidden("can not delete sso client due to permission error")
	}
}

func (kc *keycloakService) ResetServiceAccountCredentials(ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
	accessToken, _ := kc.getToken()
	orgId := auth.GetOrgIdFromContext(ctx)
	c, err := kc.getClientById(id, accessToken)
	if err != nil {
		return nil, errors.GeneralError("failed to check the service account exists:%v", err)
	}
	if isSameOrg(c, orgId) {
		credRep, error := kc.kcClient.RegenerateClientSecret(kc.ctx, accessToken, kc.config.Realm, id)
		if error != nil {
			return nil, errors.GeneralError("failed to regenerate service account secret:%v", err)
		}
		value := *credRep.Value
		return &api.ServiceAccount{
			ID:           *c.ID,
			ClientID:     *c.ClientID,
			ClientSecret: value,
			Name:         safeString(c.Name),
			Description:  safeString(c.Description),
		}, nil
	} else {
		return nil, errors.Forbidden("can not regenerate service account secret due to permission error")
	}
}

func NewUUID() string {
	return uuid.New().String()
}

func isSameOrg(client *gocloak.Client, orgId string) bool {
	if orgId == "" {
		return false
	}
	attributes := *client.Attributes
	return attributes["rh-org-id"] == orgId
}