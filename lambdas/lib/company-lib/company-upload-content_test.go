package Companylib

import (
	"bytes"
	"context"
	"encoding/base64"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/stretchr/testify/assert"
)

func TestUploadContentToS3_Base64Content(t *testing.T) {

	s3Client := &awsclients.MockS3Client{
		PutObjectOutputs: []s3.PutObjectOutput{
			{},
		},
		PutObjectErrors: []error{
			nil,
		},
	}

	svc := TenantUploadContentService{
		S3Bucket: "test-bucket",
		s3Client: s3Client,
		logger:   log.New(&bytes.Buffer{}, "", 0),
		ctx:      context.TODO(),
	}

	t.Run("It should upload content to S3 with base64 encoded data", func(t *testing.T) {
		key := "test-key"
		content := "SGVsbG8gd29ybGQh" // Base64 encoded "Hello world!"

		err := svc.UploadContentToS3_Base64Content(key, content)

		assert.NoError(t, err)
		assert.Equal(t, "application/octet-stream", *s3Client.PutObjectInputs[0].ContentType)
		assert.Equal(t, "test-bucket", *s3Client.PutObjectInputs[0].Bucket)
		assert.Equal(t, key, *s3Client.PutObjectInputs[0].Key)

		decodedContent, _ := base64.StdEncoding.DecodeString(content)
		assert.Equal(t, bytes.NewReader(decodedContent), s3Client.PutObjectInputs[0].Body)
	})

}

func TestUploadContentToS3_Base64Content_ImageType(t *testing.T) {
	s3Client := &awsclients.MockS3Client{
		PutObjectOutputs: []s3.PutObjectOutput{
			{},
		},
		PutObjectErrors: []error{
			nil,
		},
	}

	svc := TenantUploadContentService{
		S3Bucket: "test-bucket",
		s3Client: s3Client,
		logger:   log.New(&bytes.Buffer{}, "", 0),
		ctx:      context.TODO(),
	}

	t.Run("It should upload image content to S3 with base64 encoded data", func(t *testing.T) {
		key := "test-image.jpg"
		content := "SGVsbG8gd29ybGQh" // Base64 encoded "Hello world!"

		err := svc.UploadContentToS3_Base64Content(key, content)

		assert.NoError(t, err)
		assert.Equal(t, "image/jpeg", *s3Client.PutObjectInputs[0].ContentType)
		assert.Equal(t, "test-bucket", *s3Client.PutObjectInputs[0].Bucket)
		assert.Equal(t, key, *s3Client.PutObjectInputs[0].Key)

		decodedContent, _ := base64.StdEncoding.DecodeString(content)
		assert.Equal(t, bytes.NewReader(decodedContent), s3Client.PutObjectInputs[0].Body)
	})
}
