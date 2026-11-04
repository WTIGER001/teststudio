package config

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/wtiger001/teststudio/adapters/artifacts/azureblob"
	"github.com/wtiger001/teststudio/adapters/artifacts/filesystem"
	"github.com/wtiger001/teststudio/adapters/artifacts/s3"
	"github.com/wtiger001/teststudio/internal/artifacts"
)

func Open(ctx context.Context, settings Settings) (artifacts.Store, func() error, error) {
	if err := settings.Validate(); err != nil {
		return nil, nil, err
	}
	close := func() error { return nil }
	switch settings.Driver {
	case "filesystem":
		store, err := filesystem.New(settings.Directory)
		if err != nil {
			return nil, nil, err
		}
		return store, store.Close, nil
	case "s3":
		options, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(settings.S3.Region))
		if err != nil {
			return nil, nil, err
		}
		store, err := s3.New(awss3.NewFromConfig(options), settings.S3.Bucket, settings.S3.Prefix, settings.S3.ExpectedOwner, settings.S3.KMSKeyID)
		return store, close, err
	default:
		var credential azcore.TokenCredential
		var err error
		if settings.Azure.ManagedIdentityClientID != "" {
			credential, err = azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{ID: azidentity.ClientID(settings.Azure.ManagedIdentityClientID)})
		} else {
			credential, err = azidentity.NewDefaultAzureCredential(nil)
		}
		if err != nil {
			return nil, nil, err
		}
		store, err := azureblob.New(settings.Azure.ServiceURL, credential, settings.Azure.Container, settings.Azure.Prefix, nil)
		return store, close, err
	}
}
