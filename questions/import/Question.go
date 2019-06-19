package main

//Question is the model for `question` table
type Question struct {
	SurveyID     string `json:"survey_id,omitempty"`
	QuestionID   string `json:"question_id,omitempty"`
	QuestionText string `json:"question_text,omitempty"`
	Scale        string `json:"scale,omitempty"`
}
