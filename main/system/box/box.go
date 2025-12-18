package box

var Box Storage

// func Setup() (err error) {
// 	endpoint := os.Getenv("BUCKET_ENDPOINT")
// 	accesskey := os.Getenv("BUCKET_ACCESS_KEY")
// 	secretkey := os.Getenv("BUCKET_SECRET_KEY")

// 	cfg, err := config.LoadDefaultConfig(context.TODO(),
// 		config.WithBaseEndpoint(endpoint),
// 		config.WithCredentialsProvider(
// 			credentials.
// 				NewStaticCredentialsProvider(
// 					accesskey,
// 					secretkey,
// 					"",
// 				),
// 		),
// 		config.WithRegion("us-east-1"),
// 	)

// 	if err != nil {
// 		return
// 	}

// 	Box := s3.NewFromConfig(cfg)

// 	_, err = Box.CreateBucket(
// 		context.TODO(),
// 		&s3.CreateBucketInput{
// 			Bucket: aws.String("default"),
// 		},
// 	)

// 	return
//}
