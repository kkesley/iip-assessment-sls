package main

import (
	"bytes"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

//DownloadFile download a file from S3
func (app App) DownloadFile(bucket string, key string) ([]byte, error) {
	//download the content of the file that triggers the event
	s3Output, err := app.S3Service.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	//turn GetObjectOutput into []byte
	buf := new(bytes.Buffer)
	buf.ReadFrom(s3Output.Body)

	return buf.Bytes(), nil
}
