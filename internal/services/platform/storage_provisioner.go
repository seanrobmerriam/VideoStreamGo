package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	appconfig "videostreamgo/internal/config"
)

// BucketUsage holds storage usage statistics for a bucket
type BucketUsage struct {
	SizeBytes    int64     `json:"size_bytes"`
	ObjectCount  int64     `json:"object_count"`
	LastModified time.Time `json:"last_modified"`
}

// StorageProvisioner handles S3 bucket provisioning for tenant instances
type StorageProvisioner struct {
	client *s3.Client
	config *appconfig.Config
	bucket string
}

// NewStorageProvisioner creates a new storage provisioner
func NewStorageProvisioner(cfg *appconfig.Config) (*StorageProvisioner, error) {
	// Configure AWS SDK
	var awsConfig aws.Config
	var err error

	if cfg.S3.Endpoint != "" && !cfg.S3.UseSSL {
		// Use custom endpoint (MinIO or similar)
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:   "aws",
				SigningRegion: region,
				URL:           fmt.Sprintf("http://%s", cfg.S3.Endpoint),
			}, nil
		})

		awsConfig, err = awsconfig.LoadDefaultConfig(context.Background(),
			awsconfig.WithEndpointResolverWithOptions(customResolver),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.S3.AccessKey,
				cfg.S3.SecretKey,
				"",
			)),
		)
	} else {
		// Use standard AWS configuration
		awsConfig, err = awsconfig.LoadDefaultConfig(context.Background())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsConfig)

	return &StorageProvisioner{
		client: client,
		config: cfg,
		bucket: cfg.S3.Bucket,
	}, nil
}

// CreateBucket creates a new S3 bucket for a tenant instance
func (p *StorageProvisioner) CreateBucket(bucketName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Check if bucket already exists
	_, err := p.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})

	if err == nil {
		// Bucket already exists
		return nil
	}

	// Create the bucket
	createInput := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	// Set location constraint for non-us-east-1 regions
	if p.config.S3.Region != "us-east-1" {
		location := types.BucketLocationConstraint(p.config.S3.Region)
		createInput.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: location,
		}
	}

	_, err = p.client.CreateBucket(ctx, createInput)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	// Wait for bucket to be created
	waiter := s3.NewBucketExistsWaiter(p.client)
	if err := waiter.Wait(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}, 60); err != nil {
		return fmt.Errorf("bucket creation timeout: %w", err)
	}

	return nil
}

// SetBucketPolicy configures the bucket access policy
func (p *StorageProvisioner) SetBucketPolicy(ctx context.Context, bucketName string) error {
	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Sid":       "PublicReadGetObject",
				"Effect":    "Allow",
				"Principal": "*",
				"Action":    "s3:GetObject",
				"Resource":  fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
			},
		},
	}

	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	_, err = p.client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(bucketName),
		Policy: aws.String(string(policyJSON)),
	})

	return err
}

// ConfigureCORS enables CORS for video uploads from web browsers
func (p *StorageProvisioner) ConfigureCORS(ctx context.Context, bucketName string) error {
	maxAge := int32(3600)
	_, err := p.client.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: aws.String(bucketName),
		CORSConfiguration: &types.CORSConfiguration{
			CORSRules: []types.CORSRule{
				{
					AllowedHeaders: []string{"*"},
					AllowedMethods: []string{"GET", "PUT", "POST", "DELETE", "HEAD"},
					AllowedOrigins: []string{"*"},
					MaxAgeSeconds:  &maxAge,
				},
			},
		},
	})

	return err
}

// DeleteBucket removes an S3 bucket and all its contents
func (p *StorageProvisioner) DeleteBucket(ctx context.Context, bucketName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Check if bucket exists
	_, err := p.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// Bucket doesn't exist, nothing to delete
		return nil
	}

	// List and delete all objects
	if err := p.deleteAllObjects(ctx, bucketName); err != nil {
		return fmt.Errorf("failed to delete objects: %w", err)
	}

	// Delete the bucket
	_, err = p.client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	return nil
}

// deleteAllObjects removes all objects from a bucket
func (p *StorageProvisioner) deleteAllObjects(ctx context.Context, bucketName string) error {
	paginator := s3.NewListObjectsV2Paginator(p.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}

		if len(output.Contents) == 0 {
			continue
		}

		// Prepare delete request
		objects := make([]types.ObjectIdentifier, len(output.Contents))
		for i, obj := range output.Contents {
			objects[i] = types.ObjectIdentifier{
				Key: obj.Key,
			}
		}

		// Delete objects in batches
		batchSize := 1000
		for i := 0; i < len(objects); i += batchSize {
			end := i + batchSize
			if end > len(objects) {
				end = len(objects)
			}

			_, err := p.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(bucketName),
				Delete: &types.Delete{
					Objects: objects[i:end],
					Quiet:   aws.Bool(true),
				},
			})
			if err != nil {
				return fmt.Errorf("failed to delete objects: %w", err)
			}
		}
	}

	return nil
}

// GetBucketUsage returns storage usage statistics for a bucket
func (p *StorageProvisioner) GetBucketUsage(ctx context.Context, bucketName string) (*BucketUsage, error) {
	paginator := s3.NewListObjectsV2Paginator(p.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})

	usage := &BucketUsage{
		SizeBytes:   0,
		ObjectCount: 0,
	}

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range output.Contents {
			usage.SizeBytes += *obj.Size
			usage.ObjectCount++
			if obj.LastModified.After(usage.LastModified) {
				usage.LastModified = *obj.LastModified
			}
		}
	}

	return usage, nil
}

// enableVersioning enables S3 versioning for the bucket
func (p *StorageProvisioner) enableVersioning(ctx context.Context, bucketName string) error {
	status := types.BucketVersioningStatusEnabled
	_, err := p.client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucketName),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: status,
		},
	})
	return err
}

// BucketExists checks if a bucket exists
func (p *StorageProvisioner) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	_, err := p.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

// GetSignedUploadURL generates a pre-signed URL for uploading files
func (p *StorageProvisioner) GetSignedUploadURL(ctx context.Context, bucketName, key string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(p.client)

	req, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return req.URL, nil
}

// GetSignedDownloadURL generates a pre-signed URL for downloading files
func (p *StorageProvisioner) GetSignedDownloadURL(ctx context.Context, bucketName, key string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(p.client)

	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return req.URL, nil
}
