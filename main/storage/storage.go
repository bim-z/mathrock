package storage

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/charmbracelet/log"
)

var Box *s3.Client

func init() {
	endpoint := os.Getenv("BUCKET_ENDPOINT")
	accesskey := os.Getenv("BUCKET_ACCESS_KEY")
	secretkey := os.Getenv("BUCKET_SECRET_KEY")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithBaseEndpoint(endpoint),
		config.WithCredentialsProvider(
			credentials.
				NewStaticCredentialsProvider(
					accesskey,
					secretkey,
					"",
				),
		),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		log.Fatal(err)
	}

	Box := s3.NewFromConfig(cfg)

	_, err = Box.CreateBucket(
		context.TODO(),
		&s3.CreateBucketInput{
			Bucket: aws.String("default"),
		},
	)

	if err != nil {
		log.Fatal(err)
	}
}
