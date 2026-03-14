package Companylib

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

const (
	perfEntityCycle     = "CYCLE"
	perfEntityQuarter   = "QUARTER"
	perfEntityKPI       = "KPI"
	perfEntityKPIValue  = "KPI_VALUE"
	perfEntityOKR       = "OKR"
	perfEntityKeyResult = "KEY_RESULT"
	perfEntityMeeting   = "MEETING_NOTE"
	perfEntityGoalValue = "GOAL_VALUE_HISTORY"
	perfEntityGoalTeam  = "GOAL_TEAM"
	perfEntityGoalSub   = "GOAL_SUB_ITEM"
	perfEntityLadderUp  = "GOAL_LADDER_UP"
	perfEntityGoalTask  = "GOAL_TASK"
	perfGSIIndexName    = "GSI1"
	perfPKOrgPrefix     = "ORG#"
	perfSKPrefix        = "PERF#"
	perfTaskStatusTodo  = "todo"
	perfTaskStatusProg  = "in-progress"
	perfTaskStatusDone  = "completed"
)

type PerformanceService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger

	OrgPerformanceTable string
	OrganizationTable   string
}

type PerformanceRecord struct {
	PK             string                 `dynamodbav:"PK"`
	SK             string                 `dynamodbav:"SK"`
	GSI1PK         string                 `dynamodbav:"GSI1PK,omitempty"`
	GSI1SK         string                 `dynamodbav:"GSI1SK,omitempty"`
	EntityType     string                 `dynamodbav:"EntityType"`
	OrganizationId string                 `dynamodbav:"OrganizationId"`
	CycleId        string                 `dynamodbav:"CycleId,omitempty"`
	QuarterId      string                 `dynamodbav:"QuarterId,omitempty"`
	ParentId       string                 `dynamodbav:"ParentId,omitempty"`
	Owner          string                 `dynamodbav:"Owner,omitempty"`
	Status         string                 `dynamodbav:"Status,omitempty"`
	CreatedAt      string                 `dynamodbav:"CreatedAt"`
	UpdatedAt      string                 `dynamodbav:"UpdatedAt"`
	Data           map[string]interface{} `dynamodbav:"Data"`
}

type ListQueryOptions struct {
	Page     int
	PageSize int
	SortBy   string
	Order    string
}

func CreatePerformanceService(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger) *PerformanceService {
	return &PerformanceService{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
	}
}

func (svc *PerformanceService) normalizeOrgID(orgID string) string {
	if strings.HasPrefix(orgID, perfPKOrgPrefix) {
		return orgID
	}
	return perfPKOrgPrefix + orgID
}

func (svc *PerformanceService) now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (svc *PerformanceService) performanceTableName() string {
	if strings.TrimSpace(svc.OrgPerformanceTable) != "" {
		return svc.OrgPerformanceTable
	}
	return svc.OrganizationTable
}

func (svc *PerformanceService) ensurePagination(options ListQueryOptions) ListQueryOptions {
	if options.Page <= 0 {
		options.Page = 1
	}
	if options.PageSize <= 0 {
		options.PageSize = 20
	}
	if options.PageSize > 100 {
		options.PageSize = 100
	}
	if options.Order == "" {
		options.Order = "desc"
	}
	return options
}

func (svc *PerformanceService) generateID(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.NewString())
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	case string:
		parsed, err := strconv.ParseFloat(n, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func toInt(v interface{}) int {
	return int(toFloat(v))
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func (svc *PerformanceService) putRecord(record PerformanceRecord) error {
	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	_, err = svc.dynamodbClient.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.performanceTableName()),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to put record: %w", err)
	}

	return nil
}

func (svc *PerformanceService) getRecordByPKSK(orgID string, sk string) (*PerformanceRecord, error) {
	pk := svc.normalizeOrgID(orgID)
	out, err := svc.dynamodbClient.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.performanceTableName()),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}
	if out.Item == nil {
		return nil, nil
	}

	var record PerformanceRecord
	if err := attributevalue.UnmarshalMap(out.Item, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %w", err)
	}
	return &record, nil
}

func (svc *PerformanceService) getRecordByGSI1(gsi1pk string) (*PerformanceRecord, error) {
	out, err := svc.dynamodbClient.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.performanceTableName()),
		IndexName:              aws.String(perfGSIIndexName),
		KeyConditionExpression: aws.String("GSI1PK = :gsi1pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gsi1pk": &types.AttributeValueMemberS{Value: gsi1pk},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query by GSI1: %w", err)
	}
	if len(out.Items) == 0 {
		return nil, nil
	}

	var record PerformanceRecord
	if err := attributevalue.UnmarshalMap(out.Items[0], &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GSI record: %w", err)
	}
	return &record, nil
}

func (svc *PerformanceService) queryByOrgPrefix(orgID string, skPrefix string) ([]PerformanceRecord, error) {
	pk := svc.normalizeOrgID(orgID)
	out, err := svc.dynamodbClient.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.performanceTableName()),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :skPrefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":       &types.AttributeValueMemberS{Value: pk},
			":skPrefix": &types.AttributeValueMemberS{Value: skPrefix},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query by prefix: %w", err)
	}

	if len(out.Items) == 0 {
		return []PerformanceRecord{}, nil
	}

	var records []PerformanceRecord
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &records); err != nil {
		return nil, fmt.Errorf("failed to unmarshal records: %w", err)
	}
	return records, nil
}

func (svc *PerformanceService) deleteRecord(record *PerformanceRecord) error {
	_, err := svc.dynamodbClient.DeleteItem(svc.ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(svc.performanceTableName()),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: record.PK},
			"SK": &types.AttributeValueMemberS{Value: record.SK},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	return nil
}

func (svc *PerformanceService) patchRecord(record *PerformanceRecord, patch map[string]interface{}) (*PerformanceRecord, error) {
	if record.Data == nil {
		record.Data = map[string]interface{}{}
	}
	for k, v := range patch {
		record.Data[k] = v
		if k == "status" {
			record.Status = toString(v)
		}
		if k == "owner" {
			record.Owner = toString(v)
		}
	}
	record.Data["updatedAt"] = svc.now()
	record.UpdatedAt = svc.now()

	if err := svc.putRecord(*record); err != nil {
		return nil, err
	}
	return record, nil
}

func (svc *PerformanceService) toPayload(record *PerformanceRecord) map[string]interface{} {
	payload := map[string]interface{}{}
	for k, v := range record.Data {
		payload[k] = v
	}
	payload["organizationId"] = record.OrganizationId
	if record.CycleId != "" {
		payload["cycleId"] = record.CycleId
	}
	if record.QuarterId != "" {
		payload["quarterId"] = record.QuarterId
	}
	return payload
}

func (svc *PerformanceService) paginate(items []map[string]interface{}, options ListQueryOptions) (map[string]interface{}, error) {
	options = svc.ensurePagination(options)
	total := len(items)
	totalPages := 0
	if total > 0 {
		totalPages = (total + options.PageSize - 1) / options.PageSize
	}

	if options.SortBy != "" {
		sort.SliceStable(items, func(i, j int) bool {
			left := toString(items[i][options.SortBy])
			right := toString(items[j][options.SortBy])
			if strings.ToLower(options.Order) == "asc" {
				return left < right
			}
			return left > right
		})
	}

	start := (options.Page - 1) * options.PageSize
	if start > total {
		start = total
	}
	end := start + options.PageSize
	if end > total {
		end = total
	}

	result := []map[string]interface{}{}
	if start < end {
		result = items[start:end]
	}

	return map[string]interface{}{
		"items":      result,
		"total":      total,
		"page":       options.Page,
		"pageSize":   options.PageSize,
		"totalPages": totalPages,
	}, nil
}

func (svc *PerformanceService) CreatePerformanceCycle(orgID string, input map[string]interface{}) (map[string]interface{}, error) {
	cycleID := svc.generateID("cycle")
	now := svc.now()
	if input["status"] == nil || toString(input["status"]) == "" {
		input["status"] = "PLANNING"
	}
	input["id"] = cycleID
	input["organizationId"] = svc.normalizeOrgID(orgID)
	input["createdAt"] = now
	input["updatedAt"] = now

	record := PerformanceRecord{
		PK:             svc.normalizeOrgID(orgID),
		SK:             fmt.Sprintf("%sCYCLE#%s", perfSKPrefix, cycleID),
		GSI1PK:         fmt.Sprintf("%sCYCLE#%s", perfSKPrefix, cycleID),
		GSI1SK:         svc.normalizeOrgID(orgID),
		EntityType:     perfEntityCycle,
		OrganizationId: svc.normalizeOrgID(orgID),
		Status:         toString(input["status"]),
		CreatedAt:      now,
		UpdatedAt:      now,
		Data:           input,
	}

	if err := svc.putRecord(record); err != nil {
		return nil, err
	}
	return input, nil
}

func (svc *PerformanceService) ListPerformanceCycles(orgID string, filters map[string]string, options ListQueryOptions, includeQuarters bool, includeKPIs bool, includeOKRs bool) (map[string]interface{}, error) {
	records, err := svc.queryByOrgPrefix(orgID, perfSKPrefix+"CYCLE#")
	if err != nil {
		return nil, err
	}

	cycles := make([]map[string]interface{}, 0)
	for _, r := range records {
		if r.EntityType != perfEntityCycle {
			continue
		}
		data := svc.toPayload(&r)
		if filters["status"] != "" && !strings.EqualFold(toString(data["status"]), filters["status"]) {
			continue
		}
		if filters["fiscalYear"] != "" && toString(data["fiscalYear"]) != filters["fiscalYear"] {
			continue
		}
		if includeQuarters || includeKPIs || includeOKRs {
			enriched, err := svc.GetPerformanceCycleDetails(toString(data["id"]), includeQuarters, includeKPIs, includeOKRs, false)
			if err != nil {
				return nil, err
			}
			cycles = append(cycles, enriched)
		} else {
			cycles = append(cycles, data)
		}
	}

	paginated, err := svc.paginate(cycles, options)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"performanceCycles": paginated["items"],
		"total":             paginated["total"],
		"page":              paginated["page"],
		"pageSize":          paginated["pageSize"],
		"totalPages":        paginated["totalPages"],
	}, nil
}

func (svc *PerformanceService) GetPerformanceCycleDetails(cycleID string, includeQuarters bool, includeKPIs bool, includeOKRs bool, includeAnalytics bool) (map[string]interface{}, error) {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "CYCLE#" + cycleID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("performance cycle not found")
	}

	result := svc.toPayload(rec)
	relatedPrefix := fmt.Sprintf("%sCYCLE#%s#", perfSKPrefix, cycleID)
	related, err := svc.queryByOrgPrefix(rec.OrganizationId, relatedPrefix)
	if err != nil {
		return nil, err
	}

	if includeQuarters {
		quarters := make([]map[string]interface{}, 0)
		for _, item := range related {
			if item.EntityType == perfEntityQuarter {
				quarters = append(quarters, svc.toPayload(&item))
			}
		}
		result["quarters"] = quarters
	}

	if includeKPIs {
		kpis := make([]map[string]interface{}, 0)
		for _, item := range related {
			if item.EntityType == perfEntityKPI {
				kpis = append(kpis, svc.toPayload(&item))
			}
		}
		result["kpis"] = kpis
	}

	if includeOKRs {
		okrs := make([]map[string]interface{}, 0)
		for _, item := range related {
			if item.EntityType == perfEntityOKR {
				okrs = append(okrs, svc.toPayload(&item))
			}
		}
		result["okrs"] = okrs
	}

	if includeAnalytics {
		analytics, err := svc.GetCycleAnalytics(cycleID)
		if err != nil {
			return nil, err
		}
		result["analytics"] = analytics
	}

	return result, nil
}

func (svc *PerformanceService) UpdatePerformanceCycle(cycleID string, patch map[string]interface{}) (map[string]interface{}, error) {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "CYCLE#" + cycleID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("performance cycle not found")
	}
	updated, err := svc.patchRecord(rec, patch)
	if err != nil {
		return nil, err
	}
	return svc.toPayload(updated), nil
}

func (svc *PerformanceService) DeletePerformanceCycle(cycleID string) error {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "CYCLE#" + cycleID)
	if err != nil {
		return err
	}
	if rec == nil {
		return fmt.Errorf("performance cycle not found")
	}

	relatedPrefix := fmt.Sprintf("%sCYCLE#%s", perfSKPrefix, cycleID)
	related, err := svc.queryByOrgPrefix(rec.OrganizationId, relatedPrefix)
	if err != nil {
		return err
	}
	for i := range related {
		if err := svc.deleteRecord(&related[i]); err != nil {
			return err
		}
	}
	return svc.deleteRecord(rec)
}

func (svc *PerformanceService) CreateQuarter(cycleID string, input map[string]interface{}) (map[string]interface{}, error) {
	cycle, err := svc.getRecordByGSI1(perfSKPrefix + "CYCLE#" + cycleID)
	if err != nil {
		return nil, err
	}
	if cycle == nil {
		return nil, fmt.Errorf("performance cycle not found")
	}

	quarterID := svc.generateID("quarter")
	now := svc.now()
	if input["status"] == nil || toString(input["status"]) == "" {
		input["status"] = "PLANNING"
	}
	input["id"] = quarterID
	input["cycleId"] = cycleID
	input["organizationId"] = cycle.OrganizationId
	input["createdAt"] = now
	input["updatedAt"] = now

	record := PerformanceRecord{
		PK:             cycle.OrganizationId,
		SK:             fmt.Sprintf("%sCYCLE#%s#QUARTER#%s", perfSKPrefix, cycleID, quarterID),
		GSI1PK:         fmt.Sprintf("%sQUARTER#%s", perfSKPrefix, quarterID),
		GSI1SK:         fmt.Sprintf("%s#CYCLE#%s", cycle.OrganizationId, cycleID),
		EntityType:     perfEntityQuarter,
		OrganizationId: cycle.OrganizationId,
		CycleId:        cycleID,
		QuarterId:      quarterID,
		Status:         toString(input["status"]),
		CreatedAt:      now,
		UpdatedAt:      now,
		Data:           input,
	}

	if err := svc.putRecord(record); err != nil {
		return nil, err
	}
	return input, nil
}

func (svc *PerformanceService) ListQuarters(cycleID string) (map[string]interface{}, error) {
	cycle, err := svc.getRecordByGSI1(perfSKPrefix + "CYCLE#" + cycleID)
	if err != nil {
		return nil, err
	}
	if cycle == nil {
		return nil, fmt.Errorf("performance cycle not found")
	}

	related, err := svc.queryByOrgPrefix(cycle.OrganizationId, fmt.Sprintf("%sCYCLE#%s#QUARTER#", perfSKPrefix, cycleID))
	if err != nil {
		return nil, err
	}

	quarters := make([]map[string]interface{}, 0)
	for _, rec := range related {
		if rec.EntityType == perfEntityQuarter {
			quarters = append(quarters, svc.toPayload(&rec))
		}
	}
	return map[string]interface{}{"quarters": quarters}, nil
}

func (svc *PerformanceService) GetQuarterDetails(quarterID string, includeKPIs bool, includeOKRs bool, includeMeetingNotes bool, includePendingReviews bool) (map[string]interface{}, error) {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "QUARTER#" + quarterID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("quarter not found")
	}

	result := svc.toPayload(rec)
	related, err := svc.queryByOrgPrefix(rec.OrganizationId, fmt.Sprintf("%sCYCLE#%s#", perfSKPrefix, rec.CycleId))
	if err != nil {
		return nil, err
	}

	if includeKPIs {
		kpis := make([]map[string]interface{}, 0)
		for _, item := range related {
			if item.EntityType == perfEntityKPI && item.QuarterId == quarterID {
				kpis = append(kpis, svc.toPayload(&item))
			}
		}
		result["kpis"] = kpis
	}

	if includeOKRs {
		okrs := make([]map[string]interface{}, 0)
		for _, item := range related {
			if item.EntityType == perfEntityOKR && item.QuarterId == quarterID {
				okrs = append(okrs, svc.toPayload(&item))
			}
		}
		result["okrs"] = okrs
	}

	if includeMeetingNotes {
		notes := make([]map[string]interface{}, 0)
		for _, item := range related {
			if item.EntityType == perfEntityMeeting && item.QuarterId == quarterID {
				notes = append(notes, svc.toPayload(&item))
			}
		}
		result["meetingNotes"] = notes
	}

	if includePendingReviews {
		result["pendingReviews"] = []interface{}{}
	}

	return result, nil
}

func (svc *PerformanceService) UpdateQuarter(quarterID string, patch map[string]interface{}) (map[string]interface{}, error) {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "QUARTER#" + quarterID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("quarter not found")
	}
	updated, err := svc.patchRecord(rec, patch)
	if err != nil {
		return nil, err
	}
	return svc.toPayload(updated), nil
}

func (svc *PerformanceService) DeleteQuarter(quarterID string) error {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "QUARTER#" + quarterID)
	if err != nil {
		return err
	}
	if rec == nil {
		return fmt.Errorf("quarter not found")
	}
	return svc.deleteRecord(rec)
}

func (svc *PerformanceService) validateKPIInput(input map[string]interface{}) error {
	if toString(input["name"]) == "" {
		return fmt.Errorf("name is required")
	}
	if len(toString(input["name"])) > 200 {
		return fmt.Errorf("name must be <= 200 characters")
	}
	if toString(input["owner"]) == "" {
		return fmt.Errorf("owner is required")
	}
	if toString(input["status"]) != "" {
		allowed := map[string]bool{"PLANNING": true, "STARTED": true, "FINALIZED": true, "CLOSED": true}
		if !allowed[toString(input["status"])] {
			return fmt.Errorf("invalid status")
		}
	}
	if toString(input["reportingFrequency"]) != "" {
		allowed := map[string]bool{"daily": true, "weekly": true, "monthly": true, "quarterly": true, "annually": true}
		if !allowed[strings.ToLower(toString(input["reportingFrequency"]))] {
			return fmt.Errorf("invalid reportingFrequency")
		}
	}
	if input["targetValue"] == nil {
		return fmt.Errorf("targetValue is required")
	}
	green := toFloat(input["greenThreshold"])
	amber := toFloat(input["amberThreshold"])
	red := toFloat(input["redThreshold"])
	if (input["greenThreshold"] != nil || input["amberThreshold"] != nil || input["redThreshold"] != nil) && !(green >= amber && amber >= red) {
		return fmt.Errorf("greenThreshold must be >= amberThreshold >= redThreshold")
	}
	if toString(input["trend"]) != "" {
		allowed := map[string]bool{"up": true, "down": true, "stable": true}
		if !allowed[strings.ToLower(toString(input["trend"]))] {
			return fmt.Errorf("invalid trend")
		}
	}
	if toString(input["incentiveImpact"]) != "" {
		allowed := map[string]bool{"yes": true, "no": true}
		if !allowed[strings.ToLower(toString(input["incentiveImpact"]))] {
			return fmt.Errorf("invalid incentiveImpact")
		}
	}
	return nil
}

func (svc *PerformanceService) CreateKPI(input map[string]interface{}, parentKPIID string) (map[string]interface{}, error) {
	if err := svc.validateKPIInput(input); err != nil {
		return nil, err
	}

	cycleID := toString(input["cycleId"])
	if cycleID == "" {
		return nil, fmt.Errorf("cycleId is required")
	}
	cycle, err := svc.getRecordByGSI1(perfSKPrefix + "CYCLE#" + cycleID)
	if err != nil {
		return nil, err
	}
	if cycle == nil {
		return nil, fmt.Errorf("performance cycle not found")
	}

	kpiID := svc.generateID("kpi")
	now := svc.now()
	if input["status"] == nil || toString(input["status"]) == "" {
		input["status"] = "PLANNING"
	}
	input["id"] = kpiID
	input["createdAt"] = now
	input["updatedAt"] = now
	input["organizationId"] = cycle.OrganizationId
	input["cycleId"] = cycleID
	if parentKPIID != "" {
		input["parentKpiId"] = parentKPIID
	}

	quarterID := toString(input["quarterId"])
	record := PerformanceRecord{
		PK:             cycle.OrganizationId,
		SK:             fmt.Sprintf("%sCYCLE#%s#KPI#%s", perfSKPrefix, cycleID, kpiID),
		GSI1PK:         fmt.Sprintf("%sKPI#%s", perfSKPrefix, kpiID),
		GSI1SK:         fmt.Sprintf("%s#CYCLE#%s", cycle.OrganizationId, cycleID),
		EntityType:     perfEntityKPI,
		OrganizationId: cycle.OrganizationId,
		CycleId:        cycleID,
		QuarterId:      quarterID,
		ParentId:       parentKPIID,
		Owner:          toString(input["owner"]),
		Status:         toString(input["status"]),
		CreatedAt:      now,
		UpdatedAt:      now,
		Data:           input,
	}

	if err := svc.putRecord(record); err != nil {
		return nil, err
	}
	return input, nil
}

func (svc *PerformanceService) ListKPIs(orgID string, filters map[string]string, options ListQueryOptions, includeSubKPIs bool) (map[string]interface{}, error) {
	records, err := svc.queryByOrgPrefix(orgID, perfSKPrefix+"CYCLE#")
	if err != nil {
		return nil, err
	}

	kpis := make([]map[string]interface{}, 0)
	byParent := map[string][]map[string]interface{}{}
	for _, r := range records {
		if r.EntityType != perfEntityKPI {
			continue
		}
		kpi := svc.toPayload(&r)
		if filters["cycleId"] != "" && r.CycleId != filters["cycleId"] {
			continue
		}
		if filters["quarterId"] != "" && r.QuarterId != filters["quarterId"] {
			continue
		}
		if filters["department"] != "" && !strings.EqualFold(toString(kpi["department"]), filters["department"]) {
			continue
		}
		if filters["owner"] != "" && !strings.EqualFold(toString(kpi["owner"]), filters["owner"]) {
			continue
		}
		if filters["status"] != "" && !strings.EqualFold(toString(kpi["status"]), filters["status"]) {
			continue
		}
		parent := toString(kpi["parentKpiId"])
		if parent != "" {
			byParent[parent] = append(byParent[parent], kpi)
		}
		if filters["parentKpiId"] != "" && parent != filters["parentKpiId"] {
			continue
		}
		kpis = append(kpis, kpi)
	}

	if includeSubKPIs {
		for i := range kpis {
			kpis[i]["subKPIs"] = byParent[toString(kpis[i]["id"])]
		}
	}

	paginated, err := svc.paginate(kpis, options)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"kpis":       paginated["items"],
		"total":      paginated["total"],
		"page":       paginated["page"],
		"pageSize":   paginated["pageSize"],
		"totalPages": paginated["totalPages"],
	}, nil
}

func (svc *PerformanceService) GetKPIDetails(kpiID string, includeSubKPIs bool, includeValueHistory bool) (map[string]interface{}, error) {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "KPI#" + kpiID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("kpi not found")
	}

	result := svc.toPayload(rec)
	if includeSubKPIs || includeValueHistory {
		related, err := svc.queryByOrgPrefix(rec.OrganizationId, fmt.Sprintf("%sCYCLE#%s#", perfSKPrefix, rec.CycleId))
		if err != nil {
			return nil, err
		}
		if includeSubKPIs {
			subs := make([]map[string]interface{}, 0)
			for _, item := range related {
				if item.EntityType == perfEntityKPI && item.ParentId == kpiID {
					subs = append(subs, svc.toPayload(&item))
				}
			}
			result["subKPIs"] = subs
		}
		if includeValueHistory {
			history := make([]map[string]interface{}, 0)
			prefix := fmt.Sprintf("%sCYCLE#%s#KPI#%s#VALUE#", perfSKPrefix, rec.CycleId, kpiID)
			values, err := svc.queryByOrgPrefix(rec.OrganizationId, prefix)
			if err != nil {
				return nil, err
			}
			for _, value := range values {
				history = append(history, svc.toPayload(&value))
			}
			result["valueHistory"] = history
		}
	}

	return result, nil
}

func (svc *PerformanceService) UpdateKPI(kpiID string, patch map[string]interface{}) (map[string]interface{}, error) {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "KPI#" + kpiID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("kpi not found")
	}
	updated, err := svc.patchRecord(rec, patch)
	if err != nil {
		return nil, err
	}
	return svc.toPayload(updated), nil
}

func (svc *PerformanceService) DeleteKPI(kpiID string, deleteSubKPIs bool) error {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "KPI#" + kpiID)
	if err != nil {
		return err
	}
	if rec == nil {
		return fmt.Errorf("kpi not found")
	}

	if deleteSubKPIs {
		related, err := svc.queryByOrgPrefix(rec.OrganizationId, fmt.Sprintf("%sCYCLE#%s#", perfSKPrefix, rec.CycleId))
		if err != nil {
			return err
		}
		for i := range related {
			if related[i].EntityType == perfEntityKPI && related[i].ParentId == kpiID {
				if err := svc.deleteRecord(&related[i]); err != nil {
					return err
				}
			}
		}
	}

	prefix := fmt.Sprintf("%sCYCLE#%s#KPI#%s#VALUE#", perfSKPrefix, rec.CycleId, kpiID)
	values, err := svc.queryByOrgPrefix(rec.OrganizationId, prefix)
	if err == nil {
		for i := range values {
			_ = svc.deleteRecord(&values[i])
		}
	}

	return svc.deleteRecord(rec)
}

func (svc *PerformanceService) AddKPIValue(kpiID string, input map[string]interface{}, recordedBy string) (map[string]interface{}, error) {
	kpi, err := svc.getRecordByGSI1(perfSKPrefix + "KPI#" + kpiID)
	if err != nil {
		return nil, err
	}
	if kpi == nil {
		return nil, fmt.Errorf("kpi not found")
	}

	valueID := svc.generateID("kpi-value")
	now := svc.now()
	input["id"] = valueID
	input["kpiId"] = kpiID
	input["recordedBy"] = recordedBy
	input["createdAt"] = now
	if input["date"] == nil || toString(input["date"]) == "" {
		input["date"] = time.Now().UTC().Format("2006-01-02")
	}

	record := PerformanceRecord{
		PK:             kpi.PK,
		SK:             fmt.Sprintf("%sCYCLE#%s#KPI#%s#VALUE#%s", perfSKPrefix, kpi.CycleId, kpiID, valueID),
		GSI1PK:         fmt.Sprintf("%sKPI_VALUE#%s", perfSKPrefix, valueID),
		GSI1SK:         fmt.Sprintf("%s#KPI#%s", kpi.OrganizationId, kpiID),
		EntityType:     perfEntityKPIValue,
		OrganizationId: kpi.OrganizationId,
		CycleId:        kpi.CycleId,
		QuarterId:      kpi.QuarterId,
		ParentId:       kpiID,
		Owner:          recordedBy,
		CreatedAt:      now,
		UpdatedAt:      now,
		Data:           input,
	}

	if err := svc.putRecord(record); err != nil {
		return nil, err
	}

	_, _ = svc.patchRecord(kpi, map[string]interface{}{
		"currentValue": input["value"],
	})

	return input, nil
}

func (svc *PerformanceService) CreateOKR(input map[string]interface{}) (map[string]interface{}, error) {
	cycleID := toString(input["cycleId"])
	if cycleID == "" {
		return nil, fmt.Errorf("cycleId is required")
	}
	cycle, err := svc.getRecordByGSI1(perfSKPrefix + "CYCLE#" + cycleID)
	if err != nil {
		return nil, err
	}
	if cycle == nil {
		return nil, fmt.Errorf("performance cycle not found")
	}

	okrID := svc.generateID("okr")
	now := svc.now()
	if input["status"] == nil || toString(input["status"]) == "" {
		input["status"] = "DRAFT"
	}
	input["id"] = okrID
	input["createdAt"] = now
	input["updatedAt"] = now
	input["organizationId"] = cycle.OrganizationId

	record := PerformanceRecord{
		PK:             cycle.OrganizationId,
		SK:             fmt.Sprintf("%sCYCLE#%s#OKR#%s", perfSKPrefix, cycleID, okrID),
		GSI1PK:         fmt.Sprintf("%sOKR#%s", perfSKPrefix, okrID),
		GSI1SK:         fmt.Sprintf("%s#CYCLE#%s", cycle.OrganizationId, cycleID),
		EntityType:     perfEntityOKR,
		OrganizationId: cycle.OrganizationId,
		CycleId:        cycleID,
		QuarterId:      toString(input["quarterId"]),
		Owner:          toString(input["owner"]),
		Status:         toString(input["status"]),
		CreatedAt:      now,
		UpdatedAt:      now,
		Data:           input,
	}
	if err := svc.putRecord(record); err != nil {
		return nil, err
	}

	if keyResultsRaw, ok := input["keyResults"].([]interface{}); ok {
		createdKRs := make([]map[string]interface{}, 0, len(keyResultsRaw))
		for _, kr := range keyResultsRaw {
			if krMap, ok := kr.(map[string]interface{}); ok {
				created, err := svc.createKeyResult(record, krMap)
				if err != nil {
					return nil, err
				}
				createdKRs = append(createdKRs, created)
			}
		}
		input["keyResults"] = createdKRs
	}

	return input, nil
}

func (svc *PerformanceService) createKeyResult(okrRecord PerformanceRecord, kr map[string]interface{}) (map[string]interface{}, error) {
	keyResultID := svc.generateID("kr")
	now := svc.now()
	kr["id"] = keyResultID
	kr["okrId"] = toString(okrRecord.Data["id"])
	kr["createdAt"] = now
	kr["updatedAt"] = now
	if kr["status"] == nil || toString(kr["status"]) == "" {
		kr["status"] = "ON_TRACK"
	}

	record := PerformanceRecord{
		PK:             okrRecord.PK,
		SK:             fmt.Sprintf("%sCYCLE#%s#OKR#%s#KR#%s", perfSKPrefix, okrRecord.CycleId, toString(okrRecord.Data["id"]), keyResultID),
		GSI1PK:         fmt.Sprintf("%sKEYRESULT#%s", perfSKPrefix, keyResultID),
		GSI1SK:         fmt.Sprintf("%s#OKR#%s", okrRecord.OrganizationId, toString(okrRecord.Data["id"])),
		EntityType:     perfEntityKeyResult,
		OrganizationId: okrRecord.OrganizationId,
		CycleId:        okrRecord.CycleId,
		QuarterId:      okrRecord.QuarterId,
		ParentId:       toString(okrRecord.Data["id"]),
		Status:         toString(kr["status"]),
		CreatedAt:      now,
		UpdatedAt:      now,
		Data:           kr,
	}

	if err := svc.putRecord(record); err != nil {
		return nil, err
	}
	return kr, nil
}

func (svc *PerformanceService) ListOKRs(orgID string, filters map[string]string, options ListQueryOptions, includeKeyResults bool) (map[string]interface{}, error) {
	records, err := svc.queryByOrgPrefix(orgID, perfSKPrefix+"CYCLE#")
	if err != nil {
		return nil, err
	}

	okrs := make([]map[string]interface{}, 0)
	for _, r := range records {
		if r.EntityType != perfEntityOKR {
			continue
		}
		okr := svc.toPayload(&r)
		if filters["cycleId"] != "" && r.CycleId != filters["cycleId"] {
			continue
		}
		if filters["quarterId"] != "" && r.QuarterId != filters["quarterId"] {
			continue
		}
		if filters["owner"] != "" && !strings.EqualFold(toString(okr["owner"]), filters["owner"]) {
			continue
		}
		if filters["status"] != "" && !strings.EqualFold(toString(okr["status"]), filters["status"]) {
			continue
		}
		if includeKeyResults {
			krs, err := svc.getKeyResultsForOKR(toString(okr["id"]), r.OrganizationId, r.CycleId)
			if err != nil {
				return nil, err
			}
			okr["keyResults"] = krs
		}
		okrs = append(okrs, okr)
	}

	paginated, err := svc.paginate(okrs, options)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"okrs":       paginated["items"],
		"total":      paginated["total"],
		"page":       paginated["page"],
		"pageSize":   paginated["pageSize"],
		"totalPages": paginated["totalPages"],
	}, nil
}

func (svc *PerformanceService) getKeyResultsForOKR(okrID string, orgID string, cycleID string) ([]map[string]interface{}, error) {
	related, err := svc.queryByOrgPrefix(orgID, fmt.Sprintf("%sCYCLE#%s#OKR#%s#KR#", perfSKPrefix, cycleID, okrID))
	if err != nil {
		return nil, err
	}
	results := make([]map[string]interface{}, 0)
	for _, item := range related {
		if item.EntityType == perfEntityKeyResult {
			results = append(results, svc.toPayload(&item))
		}
	}
	return results, nil
}

func (svc *PerformanceService) GetOKRDetails(okrID string, includeKeyResults bool, includeProgressHistory bool) (map[string]interface{}, error) {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "OKR#" + okrID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("okr not found")
	}

	result := svc.toPayload(rec)
	if includeKeyResults {
		krs, err := svc.getKeyResultsForOKR(okrID, rec.OrganizationId, rec.CycleId)
		if err != nil {
			return nil, err
		}
		result["keyResults"] = krs
	}
	if includeProgressHistory {
		result["progressHistory"] = []interface{}{}
	}
	return result, nil
}

func (svc *PerformanceService) UpdateOKR(okrID string, patch map[string]interface{}) (map[string]interface{}, error) {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "OKR#" + okrID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("okr not found")
	}
	updated, err := svc.patchRecord(rec, patch)
	if err != nil {
		return nil, err
	}
	return svc.toPayload(updated), nil
}

func (svc *PerformanceService) DeleteOKR(okrID string) error {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "OKR#" + okrID)
	if err != nil {
		return err
	}
	if rec == nil {
		return fmt.Errorf("okr not found")
	}
	keyResults, err := svc.getKeyResultsForOKR(okrID, rec.OrganizationId, rec.CycleId)
	if err == nil {
		for _, kr := range keyResults {
			krRec, _ := svc.getRecordByGSI1(perfSKPrefix + "KEYRESULT#" + toString(kr["id"]))
			if krRec != nil {
				_ = svc.deleteRecord(krRec)
			}
		}
	}
	return svc.deleteRecord(rec)
}

func (svc *PerformanceService) UpdateKeyResult(keyResultID string, patch map[string]interface{}) (map[string]interface{}, error) {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "KEYRESULT#" + keyResultID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("key result not found")
	}
	updated, err := svc.patchRecord(rec, patch)
	if err != nil {
		return nil, err
	}
	return svc.toPayload(updated), nil
}

func (svc *PerformanceService) ListMeetingNotes(quarterID string, sortBy string, order string) (map[string]interface{}, error) {
	quarter, err := svc.getRecordByGSI1(perfSKPrefix + "QUARTER#" + quarterID)
	if err != nil {
		return nil, err
	}
	if quarter == nil {
		return nil, fmt.Errorf("quarter not found")
	}
	related, err := svc.queryByOrgPrefix(quarter.OrganizationId, fmt.Sprintf("%sCYCLE#%s#QUARTER#%s#MEETING#", perfSKPrefix, quarter.CycleId, quarterID))
	if err != nil {
		return nil, err
	}
	notes := make([]map[string]interface{}, 0)
	for _, r := range related {
		if r.EntityType == perfEntityMeeting {
			notes = append(notes, svc.toPayload(&r))
		}
	}

	if sortBy == "" {
		sortBy = "date"
	}
	if order == "" {
		order = "desc"
	}
	sort.SliceStable(notes, func(i, j int) bool {
		left := toString(notes[i][sortBy])
		right := toString(notes[j][sortBy])
		if strings.ToLower(order) == "asc" {
			return left < right
		}
		return left > right
	})

	return map[string]interface{}{"meetingNotes": notes}, nil
}

func (svc *PerformanceService) CreateMeetingNote(quarterID string, input map[string]interface{}) (map[string]interface{}, error) {
	quarter, err := svc.getRecordByGSI1(perfSKPrefix + "QUARTER#" + quarterID)
	if err != nil {
		return nil, err
	}
	if quarter == nil {
		return nil, fmt.Errorf("quarter not found")
	}

	noteID := svc.generateID("note")
	now := svc.now()
	input["id"] = noteID
	input["quarterId"] = quarterID
	input["cycleId"] = quarter.CycleId
	input["organizationId"] = quarter.OrganizationId
	input["createdAt"] = now
	input["updatedAt"] = now

	record := PerformanceRecord{
		PK:             quarter.OrganizationId,
		SK:             fmt.Sprintf("%sCYCLE#%s#QUARTER#%s#MEETING#%s", perfSKPrefix, quarter.CycleId, quarterID, noteID),
		GSI1PK:         fmt.Sprintf("%sMEETING#%s", perfSKPrefix, noteID),
		GSI1SK:         fmt.Sprintf("%s#QUARTER#%s", quarter.OrganizationId, quarterID),
		EntityType:     perfEntityMeeting,
		OrganizationId: quarter.OrganizationId,
		CycleId:        quarter.CycleId,
		QuarterId:      quarterID,
		CreatedAt:      now,
		UpdatedAt:      now,
		Data:           input,
	}
	if err := svc.putRecord(record); err != nil {
		return nil, err
	}
	return input, nil
}

func (svc *PerformanceService) UpdateMeetingNote(noteID string, patch map[string]interface{}) (map[string]interface{}, error) {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "MEETING#" + noteID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("meeting note not found")
	}
	updated, err := svc.patchRecord(rec, patch)
	if err != nil {
		return nil, err
	}
	return svc.toPayload(updated), nil
}

func (svc *PerformanceService) DeleteMeetingNote(noteID string) error {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "MEETING#" + noteID)
	if err != nil {
		return err
	}
	if rec == nil {
		return fmt.Errorf("meeting note not found")
	}
	return svc.deleteRecord(rec)
}

func (svc *PerformanceService) GetCycleAnalytics(cycleID string) (map[string]interface{}, error) {
	cycle, err := svc.getRecordByGSI1(perfSKPrefix + "CYCLE#" + cycleID)
	if err != nil {
		return nil, err
	}
	if cycle == nil {
		return nil, fmt.Errorf("performance cycle not found")
	}

	related, err := svc.queryByOrgPrefix(cycle.OrganizationId, fmt.Sprintf("%sCYCLE#%s#", perfSKPrefix, cycleID))
	if err != nil {
		return nil, err
	}

	totalKPIs := 0
	kpisOnTrack := 0
	kpisAtRisk := 0
	kpisBehind := 0
	totalOKRs := 0
	okrsCompleted := 0
	okrsOnTrack := 0
	okrsAtRisk := 0
	kpiProgressTotal := 0.0
	okrProgressTotal := 0.0
	departments := map[string][]float64{}

	for _, r := range related {
		switch r.EntityType {
		case perfEntityKPI:
			totalKPIs++
			current := toFloat(r.Data["currentValue"])
			target := toFloat(r.Data["targetValue"])
			progress := 0.0
			if target > 0 {
				progress = (current / target) * 100
			}
			kpiProgressTotal += progress
			if progress >= 90 {
				kpisOnTrack++
			} else if progress >= 60 {
				kpisAtRisk++
			} else {
				kpisBehind++
			}
			dept := toString(r.Data["department"])
			if dept != "" {
				departments[dept] = append(departments[dept], progress)
			}
		case perfEntityOKR:
			totalOKRs++
			status := strings.ToUpper(toString(r.Data["status"]))
			if status == "COMPLETED" {
				okrsCompleted++
			} else if status == "ACTIVE" {
				okrsOnTrack++
			} else if status == "DRAFT" {
				okrsAtRisk++
			}
			confidence := toFloat(r.Data["confidenceScore"])
			okrProgressTotal += confidence * 10
		}
	}

	averageKPIProgress := 0.0
	if totalKPIs > 0 {
		averageKPIProgress = kpiProgressTotal / float64(totalKPIs)
	}
	averageOKRProgress := 0.0
	if totalOKRs > 0 {
		averageOKRProgress = okrProgressTotal / float64(totalOKRs)
	}

	departmentPerf := make([]map[string]interface{}, 0)
	for dept, values := range departments {
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		avg := 0.0
		if len(values) > 0 {
			avg = sum / float64(len(values))
		}
		departmentPerf = append(departmentPerf, map[string]interface{}{
			"department":  dept,
			"kpiCount":    len(values),
			"avgProgress": avg,
		})
	}

	return map[string]interface{}{
		"cycleId": cycleID,
		"summary": map[string]interface{}{
			"totalKPIs":          totalKPIs,
			"kpisOnTrack":        kpisOnTrack,
			"kpisAtRisk":         kpisAtRisk,
			"kpisBehind":         kpisBehind,
			"totalOKRs":          totalOKRs,
			"okrsCompleted":      okrsCompleted,
			"okrsOnTrack":        okrsOnTrack,
			"okrsAtRisk":         okrsAtRisk,
			"averageKPIProgress": averageKPIProgress,
			"averageOKRProgress": averageOKRProgress,
		},
		"kpiTrends": []map[string]interface{}{
			{
				"month":   time.Now().UTC().Format("Jan 2006"),
				"onTrack": kpisOnTrack,
				"atRisk":  kpisAtRisk,
				"behind":  kpisBehind,
			},
		},
		"departmentPerformance": departmentPerf,
	}, nil
}

func (svc *PerformanceService) GetQuarterAnalytics(quarterID string) (map[string]interface{}, error) {
	quarter, err := svc.getRecordByGSI1(perfSKPrefix + "QUARTER#" + quarterID)
	if err != nil {
		return nil, err
	}
	if quarter == nil {
		return nil, fmt.Errorf("quarter not found")
	}

	cycleAnalytics, err := svc.GetCycleAnalytics(quarter.CycleId)
	if err != nil {
		return nil, err
	}
	cycleAnalytics["quarterId"] = quarterID
	return cycleAnalytics, nil
}

func (svc *PerformanceService) findGoalBase(goalID string) (map[string]interface{}, *PerformanceRecord, string, error) {
	if rec, err := svc.getRecordByGSI1(perfSKPrefix + "KPI#" + goalID); err != nil {
		return nil, nil, "", err
	} else if rec != nil {
		payload := svc.toPayload(rec)
		return payload, rec, "kpi", nil
	}
	if rec, err := svc.getRecordByGSI1(perfSKPrefix + "OKR#" + goalID); err != nil {
		return nil, nil, "", err
	} else if rec != nil {
		payload := svc.toPayload(rec)
		return payload, rec, "okr", nil
	}
	return nil, nil, "", fmt.Errorf("goal not found")
}

func (svc *PerformanceService) GetGoalDetails(goalID string, includeValueHistory bool, includeTaggedTeams bool, includeSubItems bool, includeLadderUp bool, includePrivateTasks bool, userName string) (map[string]interface{}, error) {
	base, baseRec, goalType, err := svc.findGoalBase(goalID)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"id":             goalID,
		"name":           toString(base["name"]),
		"type":           goalType,
		"description":    toString(base["description"]),
		"owner":          toString(base["owner"]),
		"currentValue":   base["currentValue"],
		"targetValue":    base["targetValue"],
		"unit":           toString(base["unitOfMeasure"]),
		"status":         toString(base["status"]),
		"deadline":       toString(base["endDate"]),
		"cycleId":        toString(base["cycleId"]),
		"quarterId":      base["quarterId"],
		"createdAt":      base["createdAt"],
		"updatedAt":      base["updatedAt"],
		"organizationId": baseRec.OrganizationId,
	}

	current := toFloat(base["currentValue"])
	target := toFloat(base["targetValue"])
	if target > 0 {
		result["progress"] = (current / target) * 100
	} else {
		result["progress"] = 0.0
	}

	if includeValueHistory {
		history, _ := svc.GetGoalValueHistory(goalID, map[string]string{}, ListQueryOptions{Page: 1, PageSize: 100})
		result["valueHistory"] = history["valueHistory"]
	}
	if includeTaggedTeams {
		teams, _ := svc.GetGoalTeams(goalID)
		result["taggedTeams"] = teams["teams"]
	}
	if includeSubItems {
		subItems, _ := svc.GetGoalSubItems(goalID)
		result["subItems"] = subItems["subItems"]
	}
	if includeLadderUp {
		ladder, _ := svc.GetGoalLadderUp(goalID, "")
		result["ladderUpItems"] = ladder["ladderUpItems"]
	}
	if includePrivateTasks {
		tasks, _ := svc.GetGoalTasks(goalID, userName, map[string]string{}, ListQueryOptions{Page: 1, PageSize: 100})
		result["privateTasks"] = tasks["tasks"]
	}

	return result, nil
}

func (svc *PerformanceService) UpdateGoal(goalID string, patch map[string]interface{}) (map[string]interface{}, error) {
	if rec, err := svc.getRecordByGSI1(perfSKPrefix + "KPI#" + goalID); err != nil {
		return nil, err
	} else if rec != nil {
		updated, err := svc.patchRecord(rec, patch)
		if err != nil {
			return nil, err
		}
		return svc.toPayload(updated), nil
	}
	if rec, err := svc.getRecordByGSI1(perfSKPrefix + "OKR#" + goalID); err != nil {
		return nil, err
	} else if rec != nil {
		updated, err := svc.patchRecord(rec, patch)
		if err != nil {
			return nil, err
		}
		return svc.toPayload(updated), nil
	}
	return nil, fmt.Errorf("goal not found")
}

func (svc *PerformanceService) GetGoalValueHistory(goalID string, filters map[string]string, options ListQueryOptions) (map[string]interface{}, error) {
	_, baseRec, _, err := svc.findGoalBase(goalID)
	if err != nil {
		return nil, err
	}

	records, err := svc.queryByOrgPrefix(baseRec.OrganizationId, fmt.Sprintf("%sGOAL#%s#VALUE#", perfSKPrefix, goalID))
	if err != nil {
		return nil, err
	}
	history := make([]map[string]interface{}, 0)
	for _, r := range records {
		entry := svc.toPayload(&r)
		date := toString(entry["date"])
		if filters["startDate"] != "" && date < filters["startDate"] {
			continue
		}
		if filters["endDate"] != "" && date > filters["endDate"] {
			continue
		}
		history = append(history, entry)
	}

	paginated, err := svc.paginate(history, options)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"valueHistory": paginated["items"],
		"total":        paginated["total"],
		"page":         paginated["page"],
		"pageSize":     paginated["pageSize"],
		"totalPages":   paginated["totalPages"],
	}, nil
}

func (svc *PerformanceService) AddGoalValueEntry(goalID string, input map[string]interface{}, userName string) (map[string]interface{}, error) {
	_, baseRec, _, err := svc.findGoalBase(goalID)
	if err != nil {
		return nil, err
	}
	entryID := svc.generateID("goal-value")
	now := svc.now()
	input["id"] = entryID
	input["goalId"] = goalID
	input["recordedBy"] = userName
	input["createdAt"] = now
	if input["date"] == nil || toString(input["date"]) == "" {
		input["date"] = time.Now().UTC().Format("2006-01-02")
	}

	record := PerformanceRecord{
		PK:             baseRec.OrganizationId,
		SK:             fmt.Sprintf("%sGOAL#%s#VALUE#%s", perfSKPrefix, goalID, entryID),
		GSI1PK:         fmt.Sprintf("%sGOAL_VALUE#%s", perfSKPrefix, entryID),
		GSI1SK:         fmt.Sprintf("%s#GOAL#%s", baseRec.OrganizationId, goalID),
		EntityType:     perfEntityGoalValue,
		OrganizationId: baseRec.OrganizationId,
		ParentId:       goalID,
		Owner:          userName,
		CreatedAt:      now,
		UpdatedAt:      now,
		Data:           input,
	}
	if err := svc.putRecord(record); err != nil {
		return nil, err
	}
	return input, nil
}

func (svc *PerformanceService) GetGoalTeams(goalID string) (map[string]interface{}, error) {
	_, baseRec, _, err := svc.findGoalBase(goalID)
	if err != nil {
		return nil, err
	}
	records, err := svc.queryByOrgPrefix(baseRec.OrganizationId, fmt.Sprintf("%sGOAL#%s#TEAM#", perfSKPrefix, goalID))
	if err != nil {
		return nil, err
	}
	teams := make([]map[string]interface{}, 0)
	for _, r := range records {
		teams = append(teams, svc.toPayload(&r))
	}
	return map[string]interface{}{"teams": teams}, nil
}

func (svc *PerformanceService) TagTeamToGoal(goalID string, input map[string]interface{}, userName string) (map[string]interface{}, error) {
	_, baseRec, _, err := svc.findGoalBase(goalID)
	if err != nil {
		return nil, err
	}
	teamID := toString(input["teamId"])
	if teamID == "" {
		return nil, fmt.Errorf("teamId is required")
	}
	now := svc.now()
	input["goalId"] = goalID
	input["taggedAt"] = now
	input["taggedBy"] = userName

	record := PerformanceRecord{
		PK:             baseRec.OrganizationId,
		SK:             fmt.Sprintf("%sGOAL#%s#TEAM#%s", perfSKPrefix, goalID, teamID),
		GSI1PK:         fmt.Sprintf("%sGOAL_TEAM#%s#%s", perfSKPrefix, goalID, teamID),
		GSI1SK:         baseRec.OrganizationId,
		EntityType:     perfEntityGoalTeam,
		OrganizationId: baseRec.OrganizationId,
		ParentId:       goalID,
		Owner:          userName,
		CreatedAt:      now,
		UpdatedAt:      now,
		Data:           input,
	}
	if err := svc.putRecord(record); err != nil {
		return nil, err
	}
	return input, nil
}

func (svc *PerformanceService) RemoveGoalTeam(goalID string, teamID string) error {
	_, baseRec, _, err := svc.findGoalBase(goalID)
	if err != nil {
		return err
	}
	record, err := svc.getRecordByPKSK(baseRec.OrganizationId, fmt.Sprintf("%sGOAL#%s#TEAM#%s", perfSKPrefix, goalID, teamID))
	if err != nil {
		return err
	}
	if record == nil {
		return fmt.Errorf("team tag not found")
	}
	return svc.deleteRecord(record)
}

// GetTeamGoals returns all OKRs and KPIs that have been tagged to a given team.
//
// It performs a reverse lookup: scans all GOAL_TEAM link records under the org
// partition and collects those whose Data["teamId"] matches the given teamID.
// For each matched goalId it calls findGoalBase to retrieve the full goal record,
// then assembles a denormalised detail map that includes computed progress.
//
// Parameters:
//   - teamID    – the team whose tagged goals are to be listed (required)
//   - orgID     – the organisation that owns the performance table partition (required)
//   - goalType  – optional filter: "kpi" or "okr" (empty string = return both)
//   - filters   – optional key/value pairs for additional filtering:
//     "status"  – e.g. "active", "completed"
//     "cycleId" – restrict results to a specific performance cycle
//   - options   – pagination / sorting options (Page, PageSize, SortBy, Order)
//
// Returns a paginated envelope:
//
//	{
//	  "goals":      []map  – array of goal detail objects
//	  "total":      int    – total goals matched (before pagination)
//	  "page":       int
//	  "pageSize":   int
//	  "totalPages": int
//	}
func (svc *PerformanceService) GetTeamGoals(teamID string, orgID string, goalType string, filters map[string]string, options ListQueryOptions) (map[string]interface{}, error) {
	// Query all GOAL_TEAM link records for this org
	records, err := svc.queryByOrgPrefix(orgID, perfSKPrefix+"GOAL#")
	if err != nil {
		return nil, err
	}

	// Collect unique goalIDs whose link record matches this team
	seen := map[string]bool{}
	goalIDs := []string{}
	for _, r := range records {
		if r.EntityType != perfEntityGoalTeam {
			continue
		}
		if toString(r.Data["teamId"]) != teamID {
			continue
		}
		gID := toString(r.Data["goalId"])
		if gID == "" || seen[gID] {
			continue
		}
		seen[gID] = true
		goalIDs = append(goalIDs, gID)
	}

	// Fetch and build the goal detail for each matched goalID
	goals := make([]map[string]interface{}, 0, len(goalIDs))
	for _, gID := range goalIDs {
		base, _, gType, err := svc.findGoalBase(gID)
		if err != nil || base == nil {
			continue
		}
		// Optional type filter: "kpi" or "okr"
		if goalType != "" && !strings.EqualFold(gType, goalType) {
			continue
		}
		// Optional status filter
		if filters["status"] != "" && !strings.EqualFold(toString(base["status"]), filters["status"]) {
			continue
		}
		// Optional cycleId filter
		if filters["cycleId"] != "" && toString(base["cycleId"]) != filters["cycleId"] {
			continue
		}
		detail := map[string]interface{}{
			"id":           gID,
			"type":         gType,
			"name":         toString(base["name"]),
			"description":  toString(base["description"]),
			"owner":        toString(base["owner"]),
			"status":       toString(base["status"]),
			"cycleId":      toString(base["cycleId"]),
			"quarterId":    base["quarterId"],
			"currentValue": base["currentValue"],
			"targetValue":  base["targetValue"],
			"unit":         toString(base["unitOfMeasure"]),
			"deadline":     toString(base["endDate"]),
			"createdAt":    base["createdAt"],
			"updatedAt":    base["updatedAt"],
		}
		current := toFloat(base["currentValue"])
		target := toFloat(base["targetValue"])
		if target > 0 {
			detail["progress"] = (current / target) * 100
		} else {
			detail["progress"] = 0.0
		}
		goals = append(goals, detail)
	}

	paginated, err := svc.paginate(goals, options)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"goals":      paginated["items"],
		"total":      paginated["total"],
		"page":       paginated["page"],
		"pageSize":   paginated["pageSize"],
		"totalPages": paginated["totalPages"],
	}, nil
}

func (svc *PerformanceService) GetGoalSubItems(goalID string) (map[string]interface{}, error) {
	_, baseRec, _, err := svc.findGoalBase(goalID)
	if err != nil {
		return nil, err
	}
	records, err := svc.queryByOrgPrefix(baseRec.OrganizationId, fmt.Sprintf("%sGOAL#%s#SUBITEM#", perfSKPrefix, goalID))
	if err != nil {
		return nil, err
	}
	subItems := make([]map[string]interface{}, 0)
	for _, r := range records {
		subItems = append(subItems, svc.toPayload(&r))
	}
	return map[string]interface{}{"subItems": subItems}, nil
}

func (svc *PerformanceService) AddGoalSubItem(goalID string, input map[string]interface{}) (map[string]interface{}, error) {
	_, baseRec, _, err := svc.findGoalBase(goalID)
	if err != nil {
		return nil, err
	}
	subItemID := svc.generateID("sub-item")
	now := svc.now()
	input["id"] = subItemID
	input["parentGoalId"] = goalID
	input["createdAt"] = now
	input["updatedAt"] = now

	record := PerformanceRecord{
		PK:             baseRec.OrganizationId,
		SK:             fmt.Sprintf("%sGOAL#%s#SUBITEM#%s", perfSKPrefix, goalID, subItemID),
		GSI1PK:         fmt.Sprintf("%sSUBITEM#%s", perfSKPrefix, subItemID),
		GSI1SK:         baseRec.OrganizationId,
		EntityType:     perfEntityGoalSub,
		OrganizationId: baseRec.OrganizationId,
		ParentId:       goalID,
		Status:         toString(input["status"]),
		CreatedAt:      now,
		UpdatedAt:      now,
		Data:           input,
	}
	if err := svc.putRecord(record); err != nil {
		return nil, err
	}
	return input, nil
}

func (svc *PerformanceService) UpdateSubItem(subItemID string, patch map[string]interface{}) (map[string]interface{}, error) {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "SUBITEM#" + subItemID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("sub-item not found")
	}
	updated, err := svc.patchRecord(rec, patch)
	if err != nil {
		return nil, err
	}
	return svc.toPayload(updated), nil
}

func (svc *PerformanceService) DeleteSubItem(subItemID string) error {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "SUBITEM#" + subItemID)
	if err != nil {
		return err
	}
	if rec == nil {
		return fmt.Errorf("sub-item not found")
	}
	return svc.deleteRecord(rec)
}

func (svc *PerformanceService) GetGoalLadderUp(goalID string, status string) (map[string]interface{}, error) {
	_, baseRec, _, err := svc.findGoalBase(goalID)
	if err != nil {
		return nil, err
	}
	records, err := svc.queryByOrgPrefix(baseRec.OrganizationId, fmt.Sprintf("%sGOAL#%s#LADDER#", perfSKPrefix, goalID))
	if err != nil {
		return nil, err
	}
	items := make([]map[string]interface{}, 0)
	for _, r := range records {
		item := svc.toPayload(&r)
		if status != "" && !strings.EqualFold(toString(item["status"]), status) {
			continue
		}
		items = append(items, item)
	}
	return map[string]interface{}{"ladderUpItems": items}, nil
}

func (svc *PerformanceService) updateLadderStatus(ladderUpID string, nextStatus string, patch map[string]interface{}) (map[string]interface{}, error) {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "LADDER#" + ladderUpID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("ladder up item not found")
	}
	patch["status"] = nextStatus
	updated, err := svc.patchRecord(rec, patch)
	if err != nil {
		return nil, err
	}
	return svc.toPayload(updated), nil
}

func (svc *PerformanceService) ApproveLadderUp(ladderUpID string, patch map[string]interface{}) (map[string]interface{}, error) {
	return svc.updateLadderStatus(ladderUpID, "APPROVED", patch)
}

func (svc *PerformanceService) RejectLadderUp(ladderUpID string, patch map[string]interface{}) (map[string]interface{}, error) {
	return svc.updateLadderStatus(ladderUpID, "REJECTED", patch)
}

func (svc *PerformanceService) GetGoalTasks(goalID string, userName string, filters map[string]string, options ListQueryOptions) (map[string]interface{}, error) {
	_, baseRec, _, err := svc.findGoalBase(goalID)
	if err != nil {
		return nil, err
	}
	records, err := svc.queryByOrgPrefix(baseRec.OrganizationId, fmt.Sprintf("%sGOAL#%s#TASK#", perfSKPrefix, goalID))
	if err != nil {
		return nil, err
	}
	tasks := make([]map[string]interface{}, 0)
	for _, r := range records {
		if r.Owner != userName {
			continue
		}
		task := svc.toPayload(&r)
		if filters["status"] != "" && !strings.EqualFold(toString(task["status"]), filters["status"]) {
			continue
		}
		tasks = append(tasks, task)
	}

	if options.SortBy == "" {
		options.SortBy = "createdAt"
	}
	paginated, err := svc.paginate(tasks, options)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"tasks":      paginated["items"],
		"total":      paginated["total"],
		"page":       paginated["page"],
		"pageSize":   paginated["pageSize"],
		"totalPages": paginated["totalPages"],
	}, nil
}

func (svc *PerformanceService) CreateGoalTask(goalID string, userName string, input map[string]interface{}) (map[string]interface{}, error) {
	_, baseRec, _, err := svc.findGoalBase(goalID)
	if err != nil {
		return nil, err
	}
	if toString(input["title"]) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if len(toString(input["title"])) > 200 {
		return nil, fmt.Errorf("title must be <= 200 characters")
	}
	status := toString(input["status"])
	if status == "" {
		status = perfTaskStatusTodo
	}
	allowed := map[string]bool{perfTaskStatusTodo: true, perfTaskStatusProg: true, perfTaskStatusDone: true}
	if !allowed[status] {
		return nil, fmt.Errorf("invalid status")
	}

	taskID := svc.generateID("task")
	now := svc.now()
	input["id"] = taskID
	input["goalId"] = goalID
	input["userId"] = userName
	input["status"] = status
	input["createdAt"] = now
	input["updatedAt"] = now
	if status == perfTaskStatusDone && toString(input["completedDate"]) == "" {
		input["completedDate"] = time.Now().UTC().Format("2006-01-02")
	}

	record := PerformanceRecord{
		PK:             baseRec.OrganizationId,
		SK:             fmt.Sprintf("%sGOAL#%s#TASK#%s#USER#%s", perfSKPrefix, goalID, taskID, userName),
		GSI1PK:         fmt.Sprintf("%sTASK#%s", perfSKPrefix, taskID),
		GSI1SK:         baseRec.OrganizationId,
		EntityType:     perfEntityGoalTask,
		OrganizationId: baseRec.OrganizationId,
		ParentId:       goalID,
		Owner:          userName,
		Status:         status,
		CreatedAt:      now,
		UpdatedAt:      now,
		Data:           input,
	}

	if err := svc.putRecord(record); err != nil {
		return nil, err
	}
	return input, nil
}

func (svc *PerformanceService) UpdateGoalTask(goalID string, taskID string, userName string, patch map[string]interface{}) (map[string]interface{}, error) {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "TASK#" + taskID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("task not found")
	}
	if rec.Owner != userName {
		return nil, fmt.Errorf("forbidden: user can only update own tasks")
	}
	if rec.ParentId != goalID {
		return nil, fmt.Errorf("task does not belong to goal")
	}
	if status := toString(patch["status"]); status == perfTaskStatusDone && toString(patch["completedDate"]) == "" {
		patch["completedDate"] = time.Now().UTC().Format("2006-01-02")
	}
	updated, err := svc.patchRecord(rec, patch)
	if err != nil {
		return nil, err
	}
	return svc.toPayload(updated), nil
}

func (svc *PerformanceService) DeleteGoalTask(goalID string, taskID string, userName string) error {
	rec, err := svc.getRecordByGSI1(perfSKPrefix + "TASK#" + taskID)
	if err != nil {
		return err
	}
	if rec == nil {
		return fmt.Errorf("task not found")
	}
	if rec.Owner != userName {
		return fmt.Errorf("forbidden: user can only delete own tasks")
	}
	if rec.ParentId != goalID {
		return fmt.Errorf("task does not belong to goal")
	}
	return svc.deleteRecord(rec)
}
