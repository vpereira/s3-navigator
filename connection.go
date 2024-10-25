package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Load saved connections from the current directory
func loadConnections() []string {
	files, err := filepath.Glob("*.connection")
	if err != nil {
		log.Printf("Error loading connections: %v", err)
		return nil
	}
	return files
}

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
