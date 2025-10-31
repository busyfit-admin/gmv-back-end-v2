package Companylib

// ------------------------------------------------------
//
// STEP FUNCTION STATES TASKS INPUT / OUTPUT Definitions - CARDS CREATION
//--------------------------------------------------------

//Step1 [Task1]:  CardsMetaData

/*
CardsMetaDataInput
{
	"trackingId" : "tracker-1",
	"cardsOrderQuantity" : 1500 ,

	"cardId" : "23445", // To be received from the Company Request - which is associated to the CardsMetatData table
	"companyId": "companyId-1"
}

CardsMetaDataOutput

{
	"trackingId" : "tracker-1",
	"cardsCreationBatches": [
		{
		"BatchId": "batch1",
		"NumberOfCards": 500,
		"cardPrefix" : "33300",
		"cardId" : "23445",
		"cardType"   : "Default", // type of card as per the CardsMetaData
		},
		{
		"BatchId": "batch2",
		"NumberOfCards": 500,
		"cardPrefix" : "33300",
		"cardId" : "23445",
		"cardType"   : "Default",
		},
		{
		"BatchId": "batch3",
		"NumberOfCards": 500,
		"cardPrefix" : "33300",
		"cardId" : "23445",
		"cardType"   : "Default",
		}
  ]
}



*/

type CardsMetaDataInput struct {
	TrackingId string `json:"TrackingId"` // Top Level Job Tracker

	CardsOrderQuantity int `json:"CardsOrderQuantity"` // Number of Cards to be Generated

	CardType  string `json:"CardType"`  // To be received from the Company Request - which is associated to the CardsMetatData table
	CardId    string `json:"CardId"`    // To be received from the Company Request - which is associated to the CardsMetatData table
	CompanyId string `json:"companyId"` // to be received from the Cognito Req auth and formatted later
}

type CardCreationBatch struct {
	TrackingId    string `json:"TrackingId"`
	BatchId       string `json:"BatchId"`
	NumberOfCards int    `json:"NumberOfCards"`
	CardPrefix    string `json:"CardPrefix"`
	CardId        string `json:"CardId"`
	CardType      string `json:"CardType"`
}

type CardsMetaDataOutput struct {
	TrackingId           string              `json:"TrackingId"`
	CardsCreationBatches []CardCreationBatch `json:"CardsCreationBatches"`
}

// Step 2 [Task2]: GenerateCardsBatch
type GenerateCardsBatchInput struct {
	TrackingId     string `json:"TrackingId"`
	BatchId        string `json:"BatchId"`
	NumberOfCards  int    `json:"NumberOfCards"`
	CardPrefix     string `json:"CardPrefix"`
	CardId         string `json:"CardId"`
	CardType       string `json:"CardType"`
	CardExpiryDate string `json:"CardExpiryDate"`
}

type GenerateCardsBatchOutput struct {
	OverallJobStatus string `json:"OverallJobStatus"`
}

// ----------------------------------------
// Common Step Function Job status
// ----------------------------------------

const (
	JOB_STATUS_STARTED        = "STARTED"
	JOB_STATUS_INPRG          = "INPROG"
	JOB_STATUS_COMPLETED      = "COMPLETED"
	JOB_STATUS_FAILED         = "FAILED"
	JOB_STATUS_INTERNAL_ERROR = "ERROR"
)

const (
	STEP_SUCCESSFULL = "SUCCESS"
	STEP_FAILED      = "FAILED"
)

const DEFAULT_CARDS_PER_BATCH = 20
