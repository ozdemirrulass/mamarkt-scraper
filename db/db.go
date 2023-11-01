package db

import (
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var db *dynamodb.DynamoDB

func Init() {
	disableSSLEnv := os.Getenv("AWS_DYNAMODB_SSL")
	disableSSLState := strings.ToLower(disableSSLEnv) == "true"

	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(os.Getenv("AWS_REGION")),
		Credentials: credentials.NewEnvCredentials(),
		Endpoint:    aws.String(os.Getenv("AWS_DYNAMODB_ENDPOINT")),
		DisableSSL:  aws.Bool(disableSSLState),
	})
	if err != nil {
		panic(err)
	}

	db = dynamodb.New(session)
}

func GetDB() *dynamodb.DynamoDB {
	return db
}
