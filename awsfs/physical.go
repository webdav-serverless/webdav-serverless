package awsfs

import (
	"context"
	"io"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type PhysicalStore struct {
	BucketName string
	S3Client   *s3.Client
}

func (s PhysicalStore) GetObject(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	result, err := s.S3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		log.Printf("Couldn't get object %v:%v. Here's why: %v\n", s.BucketName, objectKey, err)
		return nil, err
	}
	return result.Body, nil
}

func (s PhysicalStore) PutObject(ctx context.Context, objectKey string, r io.Reader) error {
	_, err := s.S3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(objectKey),
		Body:   r,
	})
	if err != nil {
		log.Printf("Couldn't upload to %v:%v. Here's why: %v\n",
			s.BucketName, objectKey, err)
	}
	return err
}

func (s PhysicalStore) PutObjectLarge(ctx context.Context, objectKey string, r io.Reader) error {
	var partMiBs int64 = 10
	uploader := manager.NewUploader(s.S3Client, func(u *manager.Uploader) {
		u.PartSize = partMiBs * 1024 * 1024
	})
	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(objectKey),
		Body:   r,
	})
	if err != nil {
		log.Printf("Couldn't upload large object to %v:%v. Here's why: %v\n",
			s.BucketName, objectKey, err)
	}
	return err
}
