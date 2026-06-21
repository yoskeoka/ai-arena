package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

const defaultArtifactPresignTTL = 5 * time.Minute

// S3ArtifactConfig describes one S3-compatible artifact lane.
type S3ArtifactConfig struct {
	Bucket          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Region          string
}

// S3ArtifactStore reads, writes, and presigns artifact objects via an S3-compatible API.
type S3ArtifactStore struct {
	bucket  string
	client  *s3.Client
	presign *s3.PresignClient
}

// NewS3ArtifactStore constructs an S3-compatible artifact store.
func NewS3ArtifactStore(ctx context.Context, cfg S3ArtifactConfig) (*S3ArtifactStore, error) {
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, fmt.Errorf("service: artifact bucket is required")
	}
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, fmt.Errorf("service: artifact endpoint is required")
	}
	if strings.TrimSpace(cfg.AccessKeyID) == "" {
		return nil, fmt.Errorf("service: artifact access key id is required")
	}
	if strings.TrimSpace(cfg.SecretAccessKey) == "" {
		return nil, fmt.Errorf("service: artifact secret access key is required")
	}
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto"
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("service: load artifact aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(cfg.Endpoint)
	})
	return &S3ArtifactStore{
		bucket:  cfg.Bucket,
		client:  client,
		presign: s3.NewPresignClient(client),
	}, nil
}

// ObjectLocator returns a stable s3:// locator for one object key.
func (s *S3ArtifactStore) ObjectLocator(key string) string {
	return fmt.Sprintf("s3://%s/%s", s.bucket, strings.TrimLeft(path.Clean(key), "/"))
}

// ReadLocator loads one object referenced by an s3:// locator.
func (s *S3ArtifactStore) ReadLocator(ctx context.Context, locator string) ([]byte, error) {
	bucket, key, err := parseS3Locator(locator)
	if err != nil {
		return nil, err
	}
	if bucket != s.bucket {
		return nil, fmt.Errorf("service: artifact bucket %q does not match configured bucket %q", bucket, s.bucket)
	}
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchKey" {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("service: get artifact object %s: %w", locator, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("service: read artifact object %s: %w", locator, err)
	}
	return data, nil
}

// PutBytes stores one object and returns its stable locator.
func (s *S3ArtifactStore) PutBytes(ctx context.Context, key string, body []byte, contentType string) (string, error) {
	key = strings.TrimLeft(path.Clean(key), "/")
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = guessContentType(key)
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	if _, err := s.client.PutObject(ctx, input); err != nil {
		return "", fmt.Errorf("service: put artifact object s3://%s/%s: %w", s.bucket, key, err)
	}
	return s.ObjectLocator(key), nil
}

// PresignGET derives a delegated download URL for one s3:// locator.
func (s *S3ArtifactStore) PresignGET(ctx context.Context, locator string, ttl time.Duration) (string, time.Time, error) {
	bucket, key, err := parseS3Locator(locator)
	if err != nil {
		return "", time.Time{}, err
	}
	if bucket != s.bucket {
		return "", time.Time{}, fmt.Errorf("service: artifact bucket %q does not match configured bucket %q", bucket, s.bucket)
	}
	if ttl <= 0 {
		ttl = defaultArtifactPresignTTL
	}
	req, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("service: presign artifact object %s: %w", locator, err)
	}
	return req.URL, time.Now().Add(ttl), nil
}

// S3ArtifactAccessIssuer derives delegated download URLs for s3:// locators.
type S3ArtifactAccessIssuer struct {
	store    *S3ArtifactStore
	fallback ArtifactAccessIssuer
	ttl      time.Duration
}

// NewS3ArtifactAccessIssuer constructs an issuer backed by S3 presigned URLs.
func NewS3ArtifactAccessIssuer(store *S3ArtifactStore) *S3ArtifactAccessIssuer {
	return &S3ArtifactAccessIssuer{
		store:    store,
		fallback: DirectArtifactAccessIssuer{},
		ttl:      defaultArtifactPresignTTL,
	}
}

// Issue derives artifact access metadata for the known locators.
func (i *S3ArtifactAccessIssuer) Issue(ctx context.Context, detail MatchDetail) (map[string]ArtifactAccessMetadata, error) {
	fallback := ArtifactAccessIssuer(DirectArtifactAccessIssuer{})
	if i != nil && i.fallback != nil {
		fallback = i.fallback
	}
	base, err := fallback.Issue(ctx, detail)
	if err != nil {
		return nil, err
	}
	if i == nil || i.store == nil {
		return base, nil
	}
	for kind, entry := range base {
		if !strings.HasPrefix(entry.Locator, "s3://") {
			continue
		}
		downloadURL, expiresAt, presignErr := i.store.PresignGET(ctx, entry.Locator, i.ttl)
		if presignErr != nil {
			return nil, presignErr
		}
		entry.DownloadURL = downloadURL
		entry.Issuer = "s3-presign"
		entry.Status = "delegated"
		entry.ExpiresAt = &expiresAt
		base[kind] = entry
	}
	return base, nil
}

func parseS3Locator(locator string) (bucket string, key string, err error) {
	parsed, err := url.Parse(strings.TrimSpace(locator))
	if err != nil {
		return "", "", fmt.Errorf("service: parse S3 locator %s: %w", locator, err)
	}
	if parsed.Scheme != "s3" {
		return "", "", fmt.Errorf("service: locator %s is not an s3:// locator", locator)
	}
	bucket = parsed.Host
	key = strings.TrimLeft(parsed.Path, "/")
	if bucket == "" || key == "" {
		return "", "", fmt.Errorf("service: invalid S3 locator %s", locator)
	}
	return bucket, key, nil
}

func guessContentType(key string) string {
	switch strings.ToLower(path.Ext(key)) {
	case ".json":
		return "application/json"
	case ".log", ".ndjson":
		return "text/plain; charset=utf-8"
	default:
		return mime.TypeByExtension(path.Ext(key))
	}
}
