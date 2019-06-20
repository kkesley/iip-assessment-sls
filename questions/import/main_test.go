package main

import (
	"testing"

	"github.com/golang/mock/gomock"
	mock_s3iface "github.com/kkesley/iip-assessment-sls/mocks/s3iface"
)

func TestHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockS3 := mock_s3iface.NewMockS3API(ctrl)
	app := App{
		S3Service: mockS3,
		ImportFn:  Import,
	}
}
