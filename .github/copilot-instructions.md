---
applyTo: "**"
---

# GMV Back-End v2 — Copilot Instructions

## Reference Files

### Employee Logic
- **Always refer to `lambdas/lib/company-lib/company-employees.go`** for any employee-related logic.
  - Use the `EmployeeService` struct as the service pattern.
  - Follow `EmployeeDynamodbData` as the canonical employee record shape.
  - DynamoDB access patterns: lookup by `UserName` (PK), `EmailId` index, `CognitoId` index, `ExternalId` index.
  - Cognito integration follows the existing patterns in that file (user creation, deletion, group assignment).
  - Team assignment and role management must follow the logic already implemented there.

### CDN / CloudFront Signing
- **Always refer to `lambdas/lib/company-lib/company-cdn-functions.go`** for any CDN or CloudFront signed-URL work.
  - Use the `CDNService` struct for all CloudFront pre-signed URL generation.
  - Private keys are stored in AWS Secrets Manager as JSON `{"private_key":"<PEM>"}` — use `AssignPrivatePublicKey()`.
  - Content signing (profile pics, images, engagements) must go through the existing methods on `CDNService`.

---

## Creating a New API

When adding a new API endpoint, three artefacts **must** be created or updated together:

### 1. API Endpoint Documentation (`docs/`)
- Create a new markdown file under `docs/102/` (or a new numbered subdirectory as appropriate).
- Follow the style of `docs/102/TEAM_GOALS_API.md`:
  - Opening section: required headers (`Authorization`, `Organization-Id`), caller permissions, date/timestamp formats, Lambda name, DynamoDB table used.
  - **Summary Table** listing all new endpoints (`#`, Method, Path, Purpose).
  - Per-endpoint sections with: Headers, Path Parameters, Query Parameters, Request Body (if applicable), Response `200` example, and error codes.
- One doc file per logical feature/module (e.g. `TEAM_GOALS_API.md`, `USER_PERFORMANCE_HUB_API.md`).

### 2. Swagger / OpenAPI Spec (`swagger-docs/tenant/tenant-apis.yaml`)
- Add every new endpoint to `swagger-docs/tenant/tenant-apis.yaml` using **OpenAPI 3.0**.
- Follow the existing conventions in that file:
  - AWS API Gateway integration block (`x-amazon-apigateway-integration`) with `type: AWS_PROXY`, `httpMethod: POST`, `passthroughBehavior: WHEN_NO_MATCH`.
  - `uri` using `Fn::Sub` to reference the Lambda ARN.
  - `security` using `- UserPool: []` for Cognito-protected endpoints.
  - Group related paths under a comment header, e.g. `# ---------- Feature Name APIs ----------`.

### 3. CloudFormation — Tenant Stack (`cfn/tenant-cfn/template.yaml`)
- **Target stack: `cfn/tenant-cfn/template.yaml` (tenant CFN only, for now).**
- For each new Lambda:
  - Resource name follows PascalCase + `Lambda` suffix (e.g. `ManageTeamOperationsLambda`).
  - `Type: AWS::Serverless::Function`
  - `Handler: bootstrap`, `Runtime: provided.al2`, `Architectures: [x86_64]`, `Timeout: 300`.
  - `CodeUri` points to the lambda directory (e.g. `../../lambdas/tenant-lambdas/<module>/<lambda-name>/`).
  - `Tracing: Active`
  - Environment variables reference CFN resources via `!Ref` / `!GetAtt`.
- Always add a corresponding `AWS::Lambda::Permission` resource named `<LambdaName>InvokePermissions` granting `apigateway.amazonaws.com` invoke access via `SourceArn: !Sub arn:aws:execute-api:...`.

---

## Updating an Existing API

When changing an existing endpoint's request/response shape, path, or behaviour:

1. **Update `docs/`** — amend the relevant markdown doc to reflect the new behaviour, parameters, or response schema.
2. **Update `swagger-docs/tenant/tenant-apis.yaml`** — keep the OpenAPI spec in sync.
3. **Update `cfn/tenant-cfn/template.yaml`** if Lambda resources, env vars, or IAM permissions change.

Never leave docs or swagger out of sync with an implementation change.

---

## Go Conventions

- Package name for shared library code: `Companylib` (see all files under `lambdas/lib/company-lib/`).
- Go module path: `github.com/busyfit-admin/saas-integrated-apis`.
- AWS SDK v2 is used throughout (`github.com/aws/aws-sdk-go-v2`).
- Client interfaces live in `lambdas/lib/clients` — use `awsclients.DynamodbClient`, `awsclients.CognitoClient`, etc.
- Logger is always `*log.Logger` passed into service constructors.
- Use `context.Context` as the first field/parameter in all services and functions.
- DynamoDB single-table design with PK/SK and GSI patterns — follow existing table designs.
- All timestamps: ISO 8601 UTC strings.
