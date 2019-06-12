package main

import (
	"bytes"
	"strings"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/google/uuid"
)

//Import questions to dynamodb table
func Import(request Request) error {
	file, err := excelize.OpenReader(bytes.NewReader(request.Content))
	if err != nil {
		return err
	}
	for _, sheet := range file.GetSheetMap() {
		surveyID := strings.TrimSpace(file.GetCellValue(sheet, "B1"))
		if len(surveyID) <= 0 {
			surveyID = uuid.New().String()
		}
	}
	return nil
}

func getExistingTemplate(request Request, surveyID string) error {
	file, err := request.App.DownloadFile(request.Bucket, "/_processed/"+surveyID+".xlsx")
	if err != nil {
		return err
	}
	return nil
}
