# Newman CLI Scripts for GMV API Testing

## Installation

First, install Newman globally:
```bash
npm install -g newman
npm install -g newman-reporter-html
```

## Basic Usage

### Run Collection with Environment
```bash
newman run postman/GMV_API_Collection.json \
  -e postman/GMV_Development_Environment.json \
  --reporters cli,json,html \
  --reporter-html-export reports/api-test-report.html \
  --reporter-json-export reports/api-test-results.json
```

### Run Specific Folder
```bash
# Test only authentication endpoints
newman run postman/GMV_API_Collection.json \
  -e postman/GMV_Development_Environment.json \
  --folder "01 - Authentication"

# Test only organization management
newman run postman/GMV_API_Collection.json \
  -e postman/GMV_Development_Environment.json \
  --folder "02 - Organization Management"
```

### Run with Data File
Create a CSV file with test data:

**test-data.csv:**
```csv
org_name,industry,company_size
"Tech Solutions Inc","Technology","11-50"
"Marketing Agency","Marketing","1-10"
"Finance Corp","Finance","51-200"
```

```bash
newman run postman/GMV_API_Collection.json \
  -e postman/GMV_Development_Environment.json \
  -d test-data.csv \
  --iteration-count 3
```

### Run with Global Variables
```bash
newman run postman/GMV_API_Collection.json \
  -e postman/GMV_Development_Environment.json \
  --global-var "api_version=v2" \
  --global-var "test_mode=true"
```

### Fail on Test Failures
```bash
newman run postman/GMV_API_Collection.json \
  -e postman/GMV_Development_Environment.json \
  --bail \
  --color on
```

## CI/CD Integration Scripts

### GitHub Actions Script
Create `.github/workflows/api-tests.yml`:

```yaml
name: API Tests
on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  api-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '18'
    
    - name: Install Newman
      run: |
        npm install -g newman
        npm install -g newman-reporter-html
    
    - name: Create reports directory
      run: mkdir -p reports
    
    - name: Run API Tests
      env:
        CLIENT_ID: ${{ secrets.COGNITO_CLIENT_ID }}
        TEST_USERNAME: ${{ secrets.TEST_USERNAME }}
        TEST_PASSWORD: ${{ secrets.TEST_PASSWORD }}
      run: |
        # Update environment file with secrets
        sed -i 's/YOUR_COGNITO_CLIENT_ID/${{ secrets.COGNITO_CLIENT_ID }}/g' postman/GMV_Development_Environment.json
        
        newman run postman/GMV_API_Collection.json \
          -e postman/GMV_Development_Environment.json \
          --reporters cli,json,html \
          --reporter-html-export reports/api-test-report.html \
          --reporter-json-export reports/api-test-results.json \
          --bail
    
    - name: Upload test reports
      uses: actions/upload-artifact@v3
      if: always()
      with:
        name: api-test-reports
        path: reports/
```

### Jenkins Pipeline Script
```groovy
pipeline {
    agent any
    
    environment {
        CLIENT_ID = credentials('cognito-client-id')
        TEST_USERNAME = credentials('test-username')
        TEST_PASSWORD = credentials('test-password')
    }
    
    stages {
        stage('Setup') {
            steps {
                sh 'npm install -g newman newman-reporter-html'
                sh 'mkdir -p reports'
            }
        }
        
        stage('API Tests') {
            steps {
                sh '''
                    newman run postman/GMV_API_Collection.json \
                      -e postman/GMV_Development_Environment.json \
                      --reporters cli,json,html \
                      --reporter-html-export reports/api-test-report.html \
                      --reporter-json-export reports/api-test-results.json \
                      --bail
                '''
            }
        }
    }
    
    post {
        always {
            publishHTML([
                allowMissing: false,
                alwaysLinkToLastBuild: true,
                keepAll: true,
                reportDir: 'reports',
                reportFiles: 'api-test-report.html',
                reportName: 'API Test Report'
            ])
            
            archiveArtifacts artifacts: 'reports/*', fingerprint: true
        }
    }
}
```

### Docker Script
Create `Dockerfile.newman`:

```dockerfile
FROM node:18-alpine

RUN npm install -g newman newman-reporter-html

WORKDIR /app

COPY postman/ postman/
COPY test-data.csv ./

CMD ["newman", "run", "postman/GMV_API_Collection.json", \
     "-e", "postman/GMV_Development_Environment.json", \
     "--reporters", "cli,json,html", \
     "--reporter-html-export", "reports/api-test-report.html"]
```

Run with Docker:
```bash
docker build -f Dockerfile.newman -t gmv-api-tests .
docker run -v $(pwd)/reports:/app/reports gmv-api-tests
```

## Advanced Usage

### Environment-Specific Testing
```bash
# Development
newman run postman/GMV_API_Collection.json \
  -e postman/GMV_Development_Environment.json

# UAT  
newman run postman/GMV_API_Collection.json \
  -e postman/GMV_UAT_Environment.json

# Production smoke tests (careful!)
newman run postman/GMV_API_Collection.json \
  -e postman/GMV_Production_Environment.json \
  --folder "06 - System Health"
```

### Parallel Execution
```bash
# Run tests in parallel using GNU parallel
echo "01 - Authentication
02 - Organization Management  
03 - Employee Management
04 - Teams Management" | parallel -j4 newman run postman/GMV_API_Collection.json \
  -e postman/GMV_Development_Environment.json \
  --folder {}
```

### Custom Reporting
```bash
newman run postman/GMV_API_Collection.json \
  -e postman/GMV_Development_Environment.json \
  --reporters cli,json,junit,html \
  --reporter-junit-export reports/junit-report.xml \
  --reporter-html-export reports/detailed-report.html \
  --reporter-json-export reports/results.json
```

### Load Testing with Newman
```bash
# Run multiple iterations to simulate load
newman run postman/GMV_API_Collection.json \
  -e postman/GMV_Development_Environment.json \
  --iteration-count 100 \
  --delay-request 500 \
  --timeout-request 10000
```

## Monitoring and Alerting

### Slack Integration
```bash
#!/bin/bash
# run-tests-with-slack.sh

newman run postman/GMV_API_Collection.json \
  -e postman/GMV_Development_Environment.json \
  --reporters cli,json \
  --reporter-json-export results.json

# Check if tests passed
if [ $? -eq 0 ]; then
    curl -X POST -H 'Content-type: application/json' \
    --data '{"text":"✅ API tests passed successfully!"}' \
    $SLACK_WEBHOOK_URL
else
    curl -X POST -H 'Content-type: application/json' \
    --data '{"text":"❌ API tests failed! Check the reports."}' \
    $SLACK_WEBHOOK_URL
fi
```

### Email Reports
```bash
#!/bin/bash
# run-tests-with-email.sh

newman run postman/GMV_API_Collection.json \
  -e postman/GMV_Development_Environment.json \
  --reporters html \
  --reporter-html-export reports/api-report.html

# Email the report
echo "API test results attached" | mail -s "GMV API Test Report" -a reports/api-report.html team@company.com
```

## Performance Monitoring

### Response Time Monitoring
```bash
newman run postman/GMV_API_Collection.json \
  -e postman/GMV_Development_Environment.json \
  --reporters json \
  --reporter-json-export results.json

# Extract performance data
node -e "
const data = require('./results.json');
console.log('Average Response Time:', 
  data.run.executions.reduce((sum, exec) => 
    sum + exec.response.responseTime, 0) / data.run.executions.length
);
"
```

This comprehensive Newman integration provides you with automated testing capabilities, CI/CD integration, and monitoring solutions for your GMV API endpoints.