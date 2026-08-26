package gcs

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"cloud.google.com/go/storage"
)

type Client struct {
	client *storage.Client
	bucket string
}

func NewClient(ctx context.Context, bucket string) (*Client, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}

	return &Client{
		client: client,
		bucket: bucket,
	}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) Bucket() string {
	return c.bucket
}

func (c *Client) GenerateUploadURL(
	objectName string,
	contentType string,
) (string, error) {

	url, err := c.client.
		Bucket(c.bucket).
		SignedURL(objectName, &storage.SignedURLOptions{
			Scheme:      storage.SigningSchemeV4,
			Method:      "PUT",
			ContentType: contentType,
			Expires:     time.Now().Add(15 * time.Minute),
		})

	if err != nil {
		return "", fmt.Errorf("generate upload signed URL: %w", err)
	}

	return url, nil
}

func (c *Client) GenerateDownloadURL(
	objectName string,
) (string, error) {

	url, err := c.client.
		Bucket(c.bucket).
		SignedURL(objectName, &storage.SignedURLOptions{
			Scheme:  storage.SigningSchemeV4,
			Method:  "GET",
			Expires: time.Now().Add(30 * time.Minute),
		})

	if err != nil {
		return "", fmt.Errorf("generate download signed URL: %w", err)
	}

	return url, nil
}

func (c *Client) Download(
	ctx context.Context,
	objectName string,
	destination string,
) error {

	reader, err := c.client.
		Bucket(c.bucket).
		Object(objectName).
		NewReader(ctx)

	if err != nil {
		return fmt.Errorf("create GCS reader: %w", err)
	}

	defer reader.Close()

	file, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}

	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("download GCS object: %w", err)
	}

	return nil
}

func (c *Client) Upload(
	ctx context.Context,
	objectName string,
	filePath string,
	contentType string,
) error {

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}

	defer file.Close()

	writer := c.client.
		Bucket(c.bucket).
		Object(objectName).
		NewWriter(ctx)

	writer.ContentType = contentType

	if _, err := io.Copy(writer, file); err != nil {
		writer.Close()
		return fmt.Errorf("upload file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close GCS writer: %w", err)
	}

	return nil
}
