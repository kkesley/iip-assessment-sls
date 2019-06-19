package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/thoas/go-funk"
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
	//query all questions for a given surveyID
	if err := app.DynamoService.QueryPages(params, queryHandler(&questions)); err != nil {
		return nil, err
	}
	return questions, nil
}

//queryHandler returns a handle for dynamodb.QueryPages in `GetQuestions`.
//it also populate *[]Question from the arguments based on the results.
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
		importSingleSheet(request, file, sheet)
	}

	//save updated excel to s3.
	return saveUpdatedExcel(request, file)
}

func saveUpdatedExcel(request Request, file *excelize.File) error {
	var buffer bytes.Buffer
	file.Write(&buffer)
	_, err := request.App.S3Service.PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(request.Bucket),
		Key:                  aws.String("__processed/" + request.Key),
		ACL:                  aws.String("private"),
		Body:                 bytes.NewReader(buffer.Bytes()),
		ContentLength:        aws.Int64(int64(len(buffer.Bytes()))),
		ContentType:          aws.String(http.DetectContentType(buffer.Bytes())),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	return err
}

//importSingleSheet process a single sheet from `Import`
func importSingleSheet(request Request, file *excelize.File, sheet string) error {
	surveyID := strings.ToUpper(strings.TrimSpace(file.GetCellValue(sheet, "B1")))
	oldQuestions := make([]Question, 0)
	newQuestions := make([]Question, 0)

	//get old questions from dynamodb
	if len(surveyID) > 0 {
		questionsOutput, err := GetQuestions(request.App, surveyID)
		if err != nil {
			return err
		}
		oldQuestions = questionsOutput
	} else {
		surveyID = uuid.New().String()
	}

	//get new questions from excel
	rows := file.GetRows(sheet)
	for rowIdx, row := range rows {
		//ignore headers and unrelated data. Row should start at index 2 (3rd row)
		if rowIdx <= 2 {
			continue
		}
		rowQuestion := Question{
			SurveyID: surveyID,
		}
		//iterate columns in a row
		for colIdx, colValue := range row {
			value := strings.TrimSpace(colValue)
			switch colIdx {
			case 0:
				rowQuestion.QuestionText = value
			case 1:
				rowQuestion.Scale = strings.ToUpper(value)
			case 2:
				rowQuestion.QuestionID = value
			}
		}
		if len(strings.TrimSpace(rowQuestion.QuestionID)) <= 0 {
			rowQuestion.QuestionID = uuid.New().String()
		}
		newQuestions = append(newQuestions, rowQuestion)
	}

	//write new question
	if err := saveNewQuestions(request, newQuestions); err != nil {
		fmt.Println(len(newQuestions))
		return err
	}

	//identify deleted questions by comparing newQuestions and oldQuestions
	questionsForDeletion := registerOldQuestionsForDeletion(oldQuestions, newQuestions)
	if len(questionsForDeletion) > 0 {
		if err := deleteUnusedQuestions(request, questionsForDeletion); err != nil {
			return err
		}
	}

	//update the excel sheet to have _ID in it.
	return updateExcelSheet(request, newQuestions, surveyID, file, sheet)
}

func updateExcelSheet(request Request, newQuestions []Question, surveyID string, file *excelize.File, sheet string) error {
	file.SetCellValue(sheet, "C2", "__ID[AUTO GENERATED.DO NOT EDIT.]")
	questionStartingRowIndex := 3
	for i, question := range newQuestions {
		file.SetCellValue(sheet, "C"+strconv.Itoa(questionStartingRowIndex+i), question.QuestionID)
	}
	return nil
}

func deleteUnusedQuestions(request Request, questionsForDeletion []Question) error {
	writeRequests := make([]*dynamodb.WriteRequest, 0)
	for _, question := range questionsForDeletion {
		writeRequest, err := dynamodbattribute.MarshalMap(Question{
			SurveyID:   question.SurveyID,
			QuestionID: question.QuestionID,
		})
		if err != nil {
			return err
		}
		writeRequests = append(writeRequests, &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: writeRequest,
			},
		})
	}
	return batchWriteDynamoDB(request, writeRequests)
}

func batchWriteDynamoDB(request Request, writeRequests []*dynamodb.WriteRequest) error {
	writeRequestsChunks := funk.Chunk(writeRequests, 25).([][]*dynamodb.WriteRequest)
	for _, chunk := range writeRequestsChunks {
		output, err := request.App.DynamoService.BatchWriteItem(&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				os.Getenv("QUESTION_TABLE"): chunk,
			},
		})
		if err != nil {
			return err
		}
		if len(output.UnprocessedItems) > 0 {
			return errors.New("There's an unprocessed item in `saveNewQuestions`")
		}
	}
	return nil
}

func saveNewQuestions(request Request, newQuestions []Question) error {
	writeRequests := make([]*dynamodb.WriteRequest, 0)
	for _, question := range newQuestions {
		writeRequest, err := dynamodbattribute.MarshalMap(question)
		if err != nil {
			return err
		}
		writeRequests = append(writeRequests, &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: writeRequest,
			},
		})
	}
	return batchWriteDynamoDB(request, funk.Shuffle(writeRequests).([]*dynamodb.WriteRequest))
}

//registerOldQuestionsForDeletion compare oldQuestions and newQuestions.
//Then give an array of questions which is a subset of oldQuestions.
//Every item in the returned array is not available in the newQuestions. Therefore, we should delete it.
func registerOldQuestionsForDeletion(oldQuestions []Question, newQuestions []Question) []Question {
	questionsForDeletion := make([]Question, 0)

	//early termination if no old questions exist
	if len(oldQuestions) <= 0 {
		return questionsForDeletion
	}

	//generate map of new question IDs to make comparison more efficient
	newQuestionIDs := make(map[string]Question)
	for _, question := range newQuestions {
		newQuestionIDs[question.QuestionID] = question
	}
	//for each old question, check whether it exists in newQuestions by comparing it to the map we just created
	for _, question := range oldQuestions {
		//if the old question does not exist in the new questions, register the questions for deletion.
		if _, ok := newQuestionIDs[question.QuestionID]; !ok {
			questionsForDeletion = append(questionsForDeletion, question)
		}
	}

	return questionsForDeletion
}
