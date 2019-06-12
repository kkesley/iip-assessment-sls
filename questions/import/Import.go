package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
)

//GetQuestions get all questions for a survey in dynamodb
func GetQuestions(app App, surveyID string) ([]Question, error) {
	params := &dynamodb.QueryInput{
		KeyConditionExpression: aws.String("#survey_id = :survey_id AND #question_id >= :question_id"),
		ExpressionAttributeNames: map[string]*string{
			"#survey_id":   aws.String("survey_id"),
			"#question_id": aws.String("question_id"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":survey_id": &dynamodb.AttributeValue{
				S: aws.String(surveyID),
			},
			":question_id": &dynamodb.AttributeValue{
				S: aws.String("_"),
			},
		},
		TableName: aws.String(os.Getenv("QUESTION_TABLE")),
	}
	questions := make([]Question, 0)
	if err := app.DynamoService.QueryPages(params, queryHandler(&questions)); err != nil {
		return nil, err
	}
	return questions, nil
}

func queryHandler(questions *[]Question) func(page *dynamodb.QueryOutput, lastPage bool) bool {
	// Unmarshal the slice of dynamodb attribute values
	// into a slice of Question structs
	return func(page *dynamodb.QueryOutput, lastPage bool) bool {
		var questionsInAPage []Question
		err := dynamodbattribute.UnmarshalListOfMaps(page.Items, &questionsInAPage)
		if err != nil {
			fmt.Printf("\nCould not unmarshal AWS data: err = %v\n", err)
			return true
		}
		//append the questions in this page to master array from arguments
		*questions = append(*questions, questionsInAPage...)

		//if lastPage is true, return false. Otherwise, return true.
		//false => end. true => continue
		return !lastPage
	}

}

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
