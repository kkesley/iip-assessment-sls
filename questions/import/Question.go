package main

//Question is the model for `question` table
type Question struct {
	SurveyID     string
	QuestionID   string
	QuestionText string
	Scale        string
}
