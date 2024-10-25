package main

import (
	"crypto/tls"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Connect to the S3 service
func connectToS3(info ConnectionInformation) error {
	customTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: info.IgnoreSSLVerification,
		},
	}
	customHTTPClient := &http.Client{Transport: customTransport}

	// Initialize MinIO client
	var err error
	minioClient, err = minio.New(info.Endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(info.AccessKey, info.SecretKey, ""),
		Secure:    true,
		Transport: customHTTPClient.Transport,
	})
	return err
}
