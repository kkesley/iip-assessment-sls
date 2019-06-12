package main

import (
	"fmt"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

//App contains dependencies for the functions
type App struct {
	S3Service s3iface.S3API
	ImportFn  func(request Request) error
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
	file, err := app.DownloadFile(record.S3.Bucket.Name, key)
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
	app := App{
		S3Service: s3.New(session.New()),
		ImportFn:  Import,
	}
	lambda.Start(app.handler)
}
