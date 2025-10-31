package Companylib

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"mime"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

type TenantUploadContentService struct {
	ctx      context.Context
	s3Client awsclients.S3Client
	logger   *log.Logger

	S3Bucket string
}

func CreateTenantUploadContentService(ctx context.Context, s3Client awsclients.S3Client, logger *log.Logger) *TenantUploadContentService {
	return &TenantUploadContentService{
		ctx:      ctx,
		s3Client: s3Client,
		logger:   logger,
	}
}

// Uploads multiple contents to S3 bucket using Base64 encoded content map with key as the file name and value as the Base64 encoded content
func (svc *TenantUploadContentService) UploadMultipleContentsToS3_Base64Content(contents map[string]string) error {
	for key, content := range contents {
		err := svc.UploadContentToS3_Base64Content(key, content)
		if err != nil {
			return err
		}
	}
	return nil
}

// Uploads content to S3 bucket using Base64 encoded content and key as the file name(file path)
func (svc *TenantUploadContentService) UploadContentToS3_Base64Content(key string, content string) error {

	// Decode the Base64 file data
	fileBytes, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		log.Printf("Failed to decode file data: %v", err)
		return fmt.Errorf("failed to decode file data %s", err)
	}

	contentType := mime.TypeByExtension(strings.ToLower(getFileExtension(key)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	putObjectInput := s3.PutObjectInput{
		Bucket:      &svc.S3Bucket,
		Key:         &key,
		Body:        bytes.NewReader(fileBytes),
		ContentType: aws.String(contentType),
	}

	_, err = svc.s3Client.PutObject(svc.ctx, &putObjectInput)
	if err != nil {
		svc.logger.Printf("Error uploading content to S3: %v\n", err)
		return err
	}
	return nil
}

// Helper functions

// getFileExtension extracts the file extension from the file name
func getFileExtension(fileName string) string {
	if idx := strings.LastIndex(fileName, "."); idx != -1 {
		return fileName[idx:]
	}
	return ""
}

// Copy an Object from source to destination in the same bucket
func (svc *TenantUploadContentService) CopyObject(sourceKey string, destinationKey string) error {
	copyObjectInput := s3.CopyObjectInput{
		Bucket:     &svc.S3Bucket,
		CopySource: aws.String(svc.S3Bucket + "/" + sourceKey),
		Key:        &destinationKey,
	}

	_, err := svc.s3Client.CopyObject(svc.ctx, &copyObjectInput)
	if err != nil {
		svc.logger.Printf("Error copying object to S3: %v\n", err)
		return err
	}
	return nil
}

// Delete an Object from S3 bucket
func (svc *TenantUploadContentService) DeleteContentFromS3(key string) error {

	svc.logger.Printf("Deleting object from S3: %s\n", key)

	deleteObjectInput := s3.DeleteObjectInput{
		Bucket: &svc.S3Bucket,
		Key:    &key,
	}

	_, err := svc.s3Client.DeleteObject(svc.ctx, &deleteObjectInput)
	if err != nil {
		svc.logger.Printf("Error deleting object from S3: %v\n", err)
		return err
	}
	return nil
}
