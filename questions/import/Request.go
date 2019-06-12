package main

//Request hold the content for performing cloning
type Request struct {
	Content []byte `json:"content"`
	Key     string `json:"key"`
}
