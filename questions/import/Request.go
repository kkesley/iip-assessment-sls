package main

//Request hold the content for performing cloning
type Request struct {
	App     App
	Bucket  string
	Content []byte `json:"content"`
	Key     string `json:"key"`
}
