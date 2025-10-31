package Companylib

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	dynamodb_attributevalue "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

type SurveyService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger

	SurveyQuestionsTable string
	SurveyResponsesTable string
}

func CreateSurveyService(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger) *SurveyService {
	return &SurveyService{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
	}
}

type SurveyQuestionsTable struct {
	SurveyId        string `json:"SurveyId" dynamodbav:"SurveyId"`
	QuestionId      string `json:"QuestionId" dynamodbav:"QuestionId"`
	QuestionText    string `json:"QuestionText" dynamodbav:"QuestionText"`
	QuestionType    string `json:"QuestionType" dynamodbav:"QuestionType"`
	QuestionOptions []string `json:"QuestionOptions" dynamodbav:"QuestionOptions"`
}

type SurveyQuestions struct {
	QuestionId      string `json:"QuestionId"`
	QuestionText    string `json:"QuestionText"`
	QuestionType    string `json:"QuestionType"`
	QuestionOptions []string `json:"QuestionOptions"`
}

type SurveyQuestionsOutput struct {
	SurveyId  string 		 `json:"SurveyId"`
	Questions []SurveyQuestions `json:"Questions"`
}

func (svc *SurveyService) GetSurveyQuestions(surveyId string) (SurveyQuestionsOutput, error) {

	getSurveyQuestionQuery := "SELECT QuestionId, QuestionText, QuestionType, QuestionOptions FROM \"" + svc.SurveyQuestionsTable + "\" WHERE SurveyId='" + surveyId + "'"

	svc.logger.Printf("Query to get the questions for the survey %s is %s", surveyId, getSurveyQuestionQuery)
	questions, err := svc.GetQuestions(getSurveyQuestionQuery)
	if err != nil {
		svc.logger.Printf("Failed to get the questions for the survey %s and failed with error : %v", surveyId, err)
		return SurveyQuestionsOutput{}, err
	}

	allQuestionsData := SurveyQuestionsOutput{}
	allQuestionsData.SurveyId = surveyId
	allQuestionsData.Questions = questions

	return allQuestionsData, nil
}

func (svc *SurveyService) GetQuestions(query string) ([]SurveyQuestions, error) {

	output, err := svc.dynamodbClient.ExecuteStatement(svc.ctx, &dynamodb.ExecuteStatementInput{
		Statement:      aws.String(query),
		ConsistentRead: aws.Bool(false),
	})

	if err != nil {
		svc.logger.Printf("Failed to run the query on DDB table and failed with error : %v", err)
		return []SurveyQuestions{}, err
	}

	if len(output.Items) == 0 {
		svc.logger.Printf("No Items found for Team Data Query %s", query)
		return []SurveyQuestions{}, nil
	}

	allQuestionsData := []SurveyQuestions{}
	for _, stageItem := range output.Items {
		questionData := SurveyQuestions{}
		err = dynamodb_attributevalue.UnmarshalMap(stageItem, &questionData)
		if err != nil {
			svc.logger.Printf("Couldn't unmarshal Team data. Failed with  Error : %v", err)
			return []SurveyQuestions{}, err
		}

		// Append data to the overall rule data
		allQuestionsData = append(allQuestionsData, questionData)
	}

	return allQuestionsData, nil
}

type SurveyResponse struct {
	QuestionId string   `json:"QuestionId"`
	Response   []string `json:"Response"`
}

type SurveyResponsesTable struct {
	ResponseId string   `json:"ResponseId" dynamodbav:"ResponseId"`
	UserName   string   `json:"UserName" dynamodbav:"UserName"`
	SurveyId   string   `json:"SurveyId" dynamodbav:"SurveyId"`
	QuestionId string   `json:"QuestionId" dynamodbav:"QuestionId"`
	Response   []string `json:"Response" dynamodbav:"Response"`
	Timestamp  string   `json:"Timestamp" dynamodbav:"Timestamp"`
}

type SurveyResponsesInput struct {
	SurveyId  string           `json:"SurveyId"`
	Responses []SurveyResponse `json:"Responses"`
}

func (svc *SurveyService) SubmitSurveyResponse(surveyResponse SurveyResponsesInput, UserName string) error {

	timestamp := utils.GenerateTimestamp()
	for _, item := range surveyResponse.Responses {
		responseId := "RESP-" + utils.GenerateRandomString(12)
		putItemInput := dynamodb.PutItemInput{
			TableName: aws.String(svc.SurveyResponsesTable),
			Item: map[string]dynamodb_types.AttributeValue{
				"ResponseId": &dynamodb_types.AttributeValueMemberS{Value: responseId},
				"UserName":   &dynamodb_types.AttributeValueMemberS{Value: UserName},
				"SurveyId":   &dynamodb_types.AttributeValueMemberS{Value: surveyResponse.SurveyId},
				"QuestionId": &dynamodb_types.AttributeValueMemberS{Value: item.QuestionId},
				"Response":   &dynamodb_types.AttributeValueMemberL{Value: convertStringSliceToAttributeValueSlice(item.Response)},
				"Timestamp":  &dynamodb_types.AttributeValueMemberS{Value: timestamp},
			},
		}

		_, err := svc.dynamodbClient.PutItem(svc.ctx, &putItemInput)
		if err != nil {
			svc.logger.Printf("Failed to put the survey response for the user %s and failed with error : %v", UserName, err)
			return err
		}
	}

	return nil
}

func convertStringSliceToAttributeValueSlice(s []string) []dynamodb_types.AttributeValue {
	attributeValues := make([]dynamodb_types.AttributeValue, len(s))
	for i, v := range s {
		attributeValues[i] = &dynamodb_types.AttributeValueMemberS{Value: v}
	}
	return attributeValues
}
