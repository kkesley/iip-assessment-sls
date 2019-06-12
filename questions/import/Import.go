package main

import (
	"bytes"

	"github.com/360EntSecGroup-Skylar/excelize"
)

//Import questions to dynamodb table
func Import(request Request) error {
	file, err := excelize.OpenReader(bytes.NewReader(request.Content))
	if err != nil {
		return err
	}
	for _, sheet := range file.GetSheetMap() {
		surveyID := file.GetCellValue(sheet, "B1")
	}
	return nil
}

func getExistingTemplate(request Request) error {
	file, err := request.App.DownloadFile(request.Bucket, "/_processed/"+request.Key)
	if err != nil {
		return err
	}
	return nil
}
