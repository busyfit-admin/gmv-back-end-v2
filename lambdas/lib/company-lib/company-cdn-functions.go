package Companylib

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	sign "github.com/aws/aws-sdk-go/service/cloudfront/sign"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

// CDNService represents the CDN service.
type CDNService struct {
	ctx    context.Context
	logger *log.Logger

	cloudfrontClient awsclients.CloudfrontClient
	secretMgrClient  awsclients.SecretManagerClient

	CDNDomain string

	publicKeyId string
	privateKey  *rsa.PrivateKey
}

// AssignPrivatePublicKey assigns the private and public keys for the CDN service.
// Private key is retrieved from AWS Secrets Manager using the provided secretKeyArn.
// format : {"private_key":"<PEM_ENCODED_PRIVATE_KEY>", "public_key":"abcdefghijklmnop"}"} , retrive only private_key from seceret string.
func (svc *CDNService) AssignPrivatePublicKey(secretKeyArn string, publicKeyId string) error {
	if secretKeyArn == "" {
		return fmt.Errorf("[ERROR] Secret ARN cannot be empty")
	}

	input := secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretKeyArn),
	}

	output, err := svc.secretMgrClient.GetSecretValue(svc.ctx, &input)
	if err != nil {
		svc.logger.Printf("[ERROR] Failed to retrieve data from the secret manager. Error : %v", err)
		return err
	}

	privateKeyString := *output.SecretString

	// get private key value from the secret string
	// assuming the secret string is in JSON format: {"private_key":"<PEM_ENCODED_PRIVATE_KEY>"}
	// we will extract the value of "private_key", extract using json package
	type secretStruct struct {
		PrivateKey string `json:"private_key"`
	}

	var secretData secretStruct
	err = json.Unmarshal([]byte(privateKeyString), &secretData)
	if err != nil {
		svc.logger.Printf("[ERROR] Failed to unmarshal secret string. Error : %v", err)
		return err
	}

	if secretData.PrivateKey == "" {
		svc.logger.Printf("[ERROR] PrivateKey Value is empty : %v ", secretData.PrivateKey)
		return fmt.Errorf("[ERROR] PrivateKey Value is empty : %v ", secretData.PrivateKey)
	}

	// Parse Private Key
	parsedPrivateKey, err := parseCDNPrivateKey(secretData.PrivateKey)
	if err != nil {
		svc.logger.Printf("[ERROR] Failed to parse Private Key : %v", err)
		return err
	}
	// Assign Private Key in Service
	svc.privateKey = parsedPrivateKey

	svc.logger.Printf("Assign Private Key Completed")

	// Assign Public Key ID of the CloudFront
	svc.publicKeyId = publicKeyId

	svc.logger.Printf("Assign Public Key Completed")

	return nil
}

// parseCDNPrivateKey parses the PEM-encoded private key.
func parseCDNPrivateKey(pemPrivateKey string) (*rsa.PrivateKey, error) {
	// Trim any leading/trailing whitespace
	pemPrivateKey = strings.TrimSpace(pemPrivateKey)

	// Parse the PEM block containing the private key
	block, _ := pem.Decode([]byte(pemPrivateKey))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing private key")
	}

	// Try PKCS1 format first (RSA PRIVATE KEY)
	if block.Type == "RSA PRIVATE KEY" {
		privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS1 private key: %v", err)
		}
		return privateKey, nil
	}

	// Try PKCS8 format (PRIVATE KEY)
	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKCS8 private key: %v", err)
	}

	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not an RSA key")
	}

	return rsaKey, nil
}

func (svc *CDNService) CreateCDNService(
	ctx context.Context,
	logger *log.Logger,
	secretMgrClient awsclients.SecretManagerClient,
	pKsSecretKeyArn string,
	publicKeyId string,

) error {

	// Assign Clients
	svc.ctx = ctx
	svc.logger = logger
	svc.secretMgrClient = secretMgrClient

	err := svc.AssignPrivatePublicKey(pKsSecretKeyArn, publicKeyId)
	if err != nil {
		svc.logger.Printf("[ERROR] Failed to Assign Private and Public Key : %v", err)
		return err
	}

	if svc.publicKeyId == "" || svc.privateKey == nil {
		return fmt.Errorf("[ERROR] Public Key ID or Private Key is empty")
	}

	signerClient := sign.NewURLSigner(svc.publicKeyId, svc.privateKey)
	svc.cloudfrontClient = signerClient

	svc.logger.Printf("[LOGGER] Assigning Clients Completed")

	return nil

}

// Gets Presigned URL for the Card present in the CloudFront Location. ( Current default expiry is 24 hours )
func (svc *CDNService) GetPreSignedCDN_URL(objectKey string) (string, error) {

	s3URL, err := GenerateDomainURL(svc.CDNDomain, objectKey)
	if err != nil {
		return "", err
	}

	singedURL, err := svc.cloudfrontClient.Sign(s3URL, time.Now().Add(1*time.Hour))

	if err != nil {
		svc.logger.Printf("Unable to sign the request")
		return "", err
	}

	return singedURL, nil
}

// GenerateURL generates an cloudfront URL for a given domain and key.
func GenerateDomainURL(domain, key string) (string, error) {
	if domain == "" || key == "" {
		return "", fmt.Errorf("domain and key must be provided")
	}

	// Format the cloudfront URL
	s3URL := fmt.Sprintf("https://%s/%s", domain, key)
	return s3URL, nil
}

// Gets Presigned URL for the Card present in the CloudFront Location. without error ( Current default expiry is 1 hours )
func (svc *CDNService) GetPreSignedCDN_URL_noError(objectKey string) string {

	s3URL, err := GenerateDomainURL(svc.CDNDomain, objectKey)
	if err != nil {
		return ""
	}

	singedURL, err := svc.cloudfrontClient.Sign(s3URL, time.Now().Add(1*time.Hour))

	if err != nil {
		svc.logger.Printf("Unable to sign the request")
		return ""
	}

	return singedURL
}

// -------------------- Engagements Module related Functions: --------------------
// SignContentInAllEngagements signs the content in all engagements.
func (svc *CDNService) SignContentInAllEngagements(allEngagements []TenantEngagementTable) []TenantEngagementTable {
	var signedEngagements []TenantEngagementTable
	for _, appreciation := range allEngagements {

		signedEngagementData := appreciation

		// Sign the Profile Pic of ProvidedByContent
		signedEngagementData.ProvidedByContent.ProfilePic = svc.GetPreSignedCDN_URL_noError(appreciation.ProvidedByContent.ProfilePic)
		// Sign the Images of Appreciation
		signedEngagementData.Images = svc.SignImages(appreciation.Images)
		// Sign the Profile Pic of RSVP Content
		signedEngagementData.RSVP = svc.SignRSVPData(appreciation.RSVP)
		// Sign the Profile Pic of Likes Content
		signedEngagementData.Likes = svc.SignLikesData(appreciation.Likes)

		// append the signed engagement data to the list
		signedEngagements = append(signedEngagements, signedEngagementData)

	}

	return signedEngagements
}

func (svc *CDNService) SignImages(images []string) []string {
	var signedImages []string
	for _, image := range images {
		signedImages = append(signedImages, svc.GetPreSignedCDN_URL_noError(image))
	}
	return signedImages
}

func (svc *CDNService) SignLikesData(likes map[string]EntityData) map[string]EntityData {
	signedLikes := make(map[string]EntityData) // Initialize the signedLikes map

	for key, like := range likes {
		like.ProfilePic = svc.GetPreSignedCDN_URL_noError(like.ProfilePic)
		if key == "" {
			continue
		}
		signedLikes[key] = like
	}

	return signedLikes
}

func (svc *CDNService) SignRSVPData(rsvp map[string]EntityData) map[string]EntityData {
	signedRSVP := make(map[string]EntityData) // Initialize the signedRSVP map

	for key, rsvp := range rsvp {
		rsvp.ProfilePic = svc.GetPreSignedCDN_URL_noError(rsvp.ProfilePic)
		if key == "" {
			continue
		}
		signedRSVP[key] = rsvp
	}

	return signedRSVP
}

/*
for _, like := range likes {
		like.ProfilePic = svc.GetPreSignedCDN_URL_noError(like.ProfilePic)
		signedLikes = append(signedLikes, like)
	}
	return signedLikes

*/
