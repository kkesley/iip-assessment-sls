package main

import (
	"bytes"
	"fmt"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func handler(event events.S3Event) error {
	if len(event.Records) < 0 {
		fmt.Println("no records")
		return nil
	}

	//get the first record of the event.
	record := event.Records[0]
	key, err := url.QueryUnescape(record.S3.Object.Key)
	if err != nil {
		return err
	}

	//download the content of the file that triggers the event
	svc := s3.New(session.New())
	s3Output, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(record.S3.Bucket.Name),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	//turn GetObjectOutput into []byte
	buf := new(bytes.Buffer)
	buf.ReadFrom(s3Output.Body)

	//perform the import
	return Import(Request{
		Content: buf.Bytes(),
		Key:     key,
	})

}

func main() {
	lambda.Start(handler)
}
