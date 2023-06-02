package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

func main() {
	// FLAGS
	var loadData bool
	var query string
	flag.BoolVar(&loadData, "load", false, "Load data into Redis")
	flag.StringVar(&query, "query", "Vaya con Dios.", "Query to search")
	flag.Parse()
	// ENV VARS
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// WEAVIATE CLIENT
	cfg := weaviate.Config{
		Host:   os.Getenv("WEAVIATE_HOST"),
		Scheme: "http",
		// Consider refresh token scenario: https://weaviate.io/developers/weaviate/client-libraries/go#-refresh-token-flow
		AuthConfig: auth.ApiKey{Value: os.Getenv("WEAVIATE_API_KEY")},
		Headers:    nil,
	}
	client, err := weaviate.NewClient(cfg)
	if err != nil {
		fmt.Println(err)
	}

	ctx := context.Background()

	if loadData {
		// CSV FILE
		quotes, err := fromCSVToQuotes("quotes.csv")
		if err != nil {
			log.Fatalf("Error reading CSV file: %v", err)
		}
		fmt.Println(quotes)
		// CREATE CLASS
		err = client.Schema().ClassCreator().WithClass(&models.Class{
			Class: "PointBreakQuote",
		}).Do(ctx)
		if err != nil {
			log.Fatalf("Error creating class: %v", err)
		}
		// BATCH OBJECTS
		objects, err := buildObjectBatch(quotes)
		if err != nil {
			log.Fatalf("Error building object batch: %v", err)
		}
		result, err := client.Batch().ObjectsBatcher().WithObjects(objects...).Do(ctx)
		if err != nil {
			log.Fatalf("Error batching objects: %v", err)
		}
		fmt.Println(result)
	}

	vec, err := vectorizeText(query)
	if err != nil {
		log.Fatalf("Error vectorizing text: %v", err)
	}

	nearVector := client.GraphQL().NearVectorArgBuilder().WithVector(vec)
	result, err := client.GraphQL().Get().
		WithClassName("PointBreakQuote").
		// WithLimit(2).
		WithFields(graphql.Field{Name: "quote"}, graphql.Field{
			Name: "_additional",
			Fields: []graphql.Field{
				{Name: "certainty"},
				{Name: "distance"},
			},
		}).
		WithNearVector(nearVector).
		Do(ctx)

	if err != nil {
		log.Fatalf("Error getting objects: %v", err)
	}
	if result.Errors != nil {
		log.Fatalf("GraphQL errors: %v", result.Errors[0].Message)
	}
	fmt.Printf("RESULT: %v", result)
}

func fromCSVToQuotes(filepath string) ([]PointBreakQuote, error) {
	csvFile, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer csvFile.Close()
	reader := csv.NewReader(csvFile)
	headers, err := reader.Read()
	fmt.Println(headers)
	if err != nil {
		return nil, err
	}
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	quotes := make([]PointBreakQuote, len(records))
	for i, record := range records {
		quotes[i] = PointBreakQuote{
			Character: record[0],
			Quote:     record[1],
		}
	}
	return quotes, nil
}

func buildObjectBatch(quotes []PointBreakQuote) ([]*models.Object, error) {
	objects := make([]*models.Object, len(quotes))
	for i, quote := range quotes {
		vector, err := vectorizeText(quote.Quote)
		if err != nil {
			return nil, err
		}
		objects[i] = &models.Object{
			Class: "PointBreakQuote",
			Properties: map[string]interface{}{
				"character": quote.Character,
				"quote":     quote.Quote,
			},
			Vector: vector,
		}
	}
	return objects, nil
}
