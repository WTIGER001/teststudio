package azureblob

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/wtiger001/teststudio/internal/artifacts"
)

type Store struct {
	service           *url.URL
	credential        azcore.TokenCredential
	options           *blockblob.ClientOptions
	container, prefix string
}

func New(serviceURL string, credential azcore.TokenCredential, container, prefix string, options *blockblob.ClientOptions) (*Store, error) {
	base, err := url.Parse(serviceURL)
	if err != nil || base.Scheme != "https" || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || credential == nil || container == "" || !artifacts.ValidPrefix(prefix) {
		return nil, errors.New("Azure Blob requires an HTTPS origin, credential, container, and canonical prefix")
	}
	return &Store{base, credential, options, container, prefix}, nil
}

func (s *Store) client(key string) (*blockblob.Client, error) {
	objectKey, err := artifacts.ObjectKey(s.prefix, key)
	if err != nil {
		return nil, err
	}
	endpoint := *s.service
	endpoint.Path = strings.TrimSuffix(endpoint.Path, "/") + "/" + s.container + "/" + objectKey
	endpoint.RawPath = ""
	return blockblob.NewClient(endpoint.String(), s.credential, s.options)
}

func (s *Store) Put(ctx context.Context, key string, source io.Reader, limit int64) (artifacts.Object, error) {
	client, err := s.client(key)
	if err != nil {
		return artifacts.Object{}, err
	}
	staged, err := artifacts.Stage(ctx, source, limit)
	if err != nil {
		return artifacts.Object{}, err
	}
	defer staged.Close()
	contentType, cacheControl := "application/octet-stream", "private, no-store"
	_, err = client.UploadFile(ctx, staged.File, &blockblob.UploadFileOptions{HTTPHeaders: &blob.HTTPHeaders{BlobContentType: &contentType, BlobCacheControl: &cacheControl}, BlockSize: 4 << 20, TransactionalValidation: blob.TransferValidationTypeComputeCRC64(), Metadata: map[string]*string{"teststudio_sha256": &staged.Object.Checksum}, Concurrency: 1})
	if err != nil {
		return artifacts.Object{}, err
	}
	return staged.Object, nil
}

func (s *Store) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	client, err := s.client(key)
	if err != nil {
		return nil, err
	}
	response, err := client.DownloadStream(ctx, nil)
	if bloberror.HasCode(err, bloberror.BlobNotFound) {
		return nil, artifacts.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return response.NewRetryReader(ctx, nil), nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	client, err := s.client(key)
	if err != nil {
		return err
	}
	_, err = client.Delete(ctx, nil)
	if bloberror.HasCode(err, bloberror.BlobNotFound) {
		return nil
	}
	return err
}
