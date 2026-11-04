package s3

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/wtiger001/teststudio/internal/artifacts"
)

type Store struct {
	client                     *awss3.Client
	bucket, prefix, owner, kms string
}

func New(client *awss3.Client, bucket, prefix, owner, kms string) (*Store, error) {
	if client == nil || bucket == "" || !artifacts.ValidPrefix(prefix) {
		return nil, errors.New("S3 requires a client, bucket, and canonical object prefix")
	}
	return &Store{client, bucket, prefix, owner, kms}, nil
}

func (s *Store) expectedOwner() *string {
	if s.owner == "" {
		return nil
	}
	return aws.String(s.owner)
}

func (s *Store) Put(ctx context.Context, key string, source io.Reader, limit int64) (artifacts.Object, error) {
	objectKey, err := artifacts.ObjectKey(s.prefix, key)
	if err != nil {
		return artifacts.Object{}, err
	}
	staged, err := artifacts.Stage(ctx, source, limit)
	if err != nil {
		return artifacts.Object{}, err
	}
	defer staged.Close()
	hash, _ := hex.DecodeString(staged.Object.Checksum)
	checksum := base64.StdEncoding.EncodeToString(hash)
	input := &awss3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(objectKey), Body: staged.File, ContentLength: aws.Int64(staged.Object.Size), ContentType: aws.String("application/octet-stream"), CacheControl: aws.String("private, no-store"), ChecksumSHA256: aws.String(checksum), Metadata: map[string]string{"teststudio-sha256": staged.Object.Checksum}, ExpectedBucketOwner: s.expectedOwner()}
	if s.kms != "" {
		input.ServerSideEncryption = types.ServerSideEncryptionAwsKms
		input.SSEKMSKeyId = aws.String(s.kms)
	}
	result, err := s.client.PutObject(ctx, input)
	if err != nil {
		return artifacts.Object{}, err
	}
	if result.ChecksumSHA256 != nil && *result.ChecksumSHA256 != checksum {
		return artifacts.Object{}, errors.New("S3 upload checksum does not match captured evidence")
	}
	return staged.Object, nil
}

func missing(err error) bool {
	var api smithy.APIError
	return errors.As(err, &api) && (api.ErrorCode() == "NoSuchKey" || api.ErrorCode() == "NotFound")
}

func (s *Store) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	objectKey, err := artifacts.ObjectKey(s.prefix, key)
	if err != nil {
		return nil, err
	}
	response, err := s.client.GetObject(ctx, &awss3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(objectKey), ExpectedBucketOwner: s.expectedOwner()})
	if missing(err) {
		return nil, artifacts.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	objectKey, err := artifacts.ObjectKey(s.prefix, key)
	if err != nil {
		return err
	}
	_, err = s.client.DeleteObject(ctx, &awss3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(objectKey), ExpectedBucketOwner: s.expectedOwner()})
	if missing(err) {
		return nil
	}
	return err
}
