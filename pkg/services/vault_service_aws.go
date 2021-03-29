package services

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
)

var OwnerResourceTagKey = "owner-resource"

var _ VaultService = &awsVaultService{}

type awsVaultService struct {
	secretCache  *secretcache.Cache
	secretClient *secretsmanager.SecretsManager
}

func NewAwsVaultService(vaultConfig *config.VaultConfig) (*awsVaultService, error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(
			vaultConfig.AccessKey,
			vaultConfig.SecretAccessKey,
			""),
		//Region:  aws.String(region),
		Retryer: client.DefaultRetryer{NumMaxRetries: 2},
	})
	if err != nil {
		return nil, err
	}

	secretClient := secretsmanager.New(sess)
	secretCache, err := secretcache.New(func(cache *secretcache.Cache) {
		cache.Client = secretClient
	})
	if err != nil {
		return nil, err
	}
	return &awsVaultService{
		secretClient: secretClient,
		secretCache:  secretCache,
	}, nil
}

func (k *awsVaultService) Kind() string {
	return "aws"
}

func (k *awsVaultService) GetSecretString(name string) (string, error) {
	return k.secretCache.GetSecretString(name)
}

func (k *awsVaultService) SetSecretString(name string, value string, owningResource string) error {
	_, err := k.secretClient.PutSecretValue(&secretsmanager.PutSecretValueInput{
		SecretId:     &name,
		SecretString: &value,
	})
	if err != nil {
		return err
	}

	if owningResource != "" {
		_, err = k.secretClient.TagResource(&secretsmanager.TagResourceInput{
			SecretId: &name,
			Tags: []*secretsmanager.Tag{
				&secretsmanager.Tag{
					Key:   &OwnerResourceTagKey,
					Value: &owningResource,
				},
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *awsVaultService) ForEachSecret(f func(name string, owningResource string) bool) error {
	paging := &secretsmanager.ListSecretsInput{}
	err := k.secretClient.ListSecretsPages(paging, func(output *secretsmanager.ListSecretsOutput, lastPage bool) bool {
		for _, entry := range output.SecretList {
			owner := getTag(entry.Tags, OwnerResourceTagKey)
			name := ""
			if entry.Name != nil {
				name = *entry.Name
			}
			if !f(name, owner) {
				return false
			}
		}
		return false
	})
	if err != nil {
		return err
	}
	return nil
}

func getTag(tags []*secretsmanager.Tag, key string) string {
	for _, tag := range tags {
		if *tag.Key == key {
			return *tag.Value
		}
	}
	return ""
}

func (k *awsVaultService) DeleteSecretString(name string) error {
	_, err := k.secretClient.DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId: &name,
	})
	return err
}