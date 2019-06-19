package main

import (
	"bytes"
	"fmt"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

//App contains dependencies for the functions
type App struct {
	S3Service     s3iface.S3API
	DynamoService dynamodbiface.DynamoDBAPI
	ImportFn      func(request Request) error
}

func downloadFile(svc s3iface.S3API, bucket string, key string) ([]byte, error) {
	//download the content of the file that triggers the event
	s3Output, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	fmt.Println(bucket)
	fmt.Println(key)
	if err != nil {
		fmt.Println("failed do download s3")
		return nil, err
	}

	//turn GetObjectOutput into []byte
	buf := new(bytes.Buffer)
	buf.ReadFrom(s3Output.Body)

	return buf.Bytes(), nil
}

func (app App) handler(event events.S3Event) error {
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
	file, err := downloadFile(app.S3Service, record.S3.Bucket.Name, key)
	if err != nil {
		return err
	}

	//perform the import
	{
		err := app.ImportFn(Request{
			App:     app,
			Bucket:  record.S3.Bucket.Name,
			Content: file,
			Key:     key,
		})
		if err != nil {
			fmt.Println(err) //log the error
		}
		return err
	}
}

func main() {
	sess := session.New()
	app := App{
		S3Service:     s3.New(sess),
		DynamoService: dynamodb.New(sess),
		ImportFn:      Import,
	}
	lambda.Start(app.handler)
}
