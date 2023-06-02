package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

var modelId string = "sentence-transformers/all-MiniLM-L6-v2"
var hg_api string = fmt.Sprintf("https://api-inference.huggingface.co/pipeline/feature-extraction/%s", modelId)

type FeatureExtractionOptions struct {
	UseCache     bool `json:"use_cache"`
	WaitForModel bool `json:"wait_for_model"`
}

type FeatureExtractionPayload struct {
	Inputs  string                   `json:"inputs"`
	Options FeatureExtractionOptions `json:"options"`
}

type PointBreakQuote struct {
	Character string
	Quote     string
	Vector    []float32
}

func vectorizeText(inputs string) ([]float32, error) {
	payload := FeatureExtractionPayload{
		Inputs: inputs,
		Options: FeatureExtractionOptions{
			UseCache:     true,
			WaitForModel: true,
		},
	}

	jsonStr, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload: %v", err)
	}

	req, err := http.NewRequest("POST", hg_api, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("HG_TOKEN")))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("Hugging Face Feature Extraction Response Status:", resp.Status)

	var data []float32
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("error decoding response body: %v", err)
	}

	return data, nil
}
