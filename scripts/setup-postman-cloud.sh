#!/bin/bash

# Postman Cloud Integration Setup Script
# This script helps you set up automated API testing with Postman Cloud

set -e

echo "🚀 GMV Postman Cloud Integration Setup"
echo "======================================"

# Check if required tools are installed
check_dependencies() {
    echo "📋 Checking dependencies..."
    
    if ! command -v curl &> /dev/null; then
        echo "❌ curl is required but not installed"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        echo "❌ jq is required but not installed"
        echo "📥 Install with: sudo apt-get install jq (Ubuntu/Debian) or brew install jq (macOS)"
        exit 1
    fi
    
    echo "✅ Dependencies check passed"
}

# Get Postman API key
get_postman_api_key() {
    echo ""
    echo "🔐 Postman API Key Setup"
    echo "------------------------"
    echo "1. Go to https://go.postman.co/settings/me/api-keys"
    echo "2. Click 'Generate API Key'"
    echo "3. Name it 'GitHub Actions Integration'"
    echo "4. Copy the generated key"
    echo ""
    
    read -s -p "Enter your Postman API Key: " POSTMAN_API_KEY
    echo ""
    
    if [ -z "$POSTMAN_API_KEY" ]; then
        echo "❌ API Key cannot be empty"
        exit 1
    fi
    
    # Test the API key
    echo "🧪 Testing API key..."
    RESPONSE=$(curl -s -w "%{http_code}" -X GET "https://api.getpostman.com/me" \
        -H "X-API-Key: $POSTMAN_API_KEY" -o /tmp/postman_test.json)
    
    if [ "$RESPONSE" -eq 200 ]; then
        USER_NAME=$(cat /tmp/postman_test.json | jq -r '.user.fullName // .user.username')
        echo "✅ API Key valid - Connected as: $USER_NAME"
        rm /tmp/postman_test.json
    else
        echo "❌ Invalid API Key (HTTP $RESPONSE)"
        exit 1
    fi
}

# List and select workspace
select_workspace() {
    echo ""
    echo "🏢 Workspace Selection"
    echo "----------------------"
    
    echo "📋 Fetching your workspaces..."
    WORKSPACES=$(curl -s -X GET "https://api.getpostman.com/workspaces" \
        -H "X-API-Key: $POSTMAN_API_KEY")
    
    echo "$WORKSPACES" | jq -r '.workspaces[] | "ID: \(.id) | Name: \(.name) | Type: \(.type)"' | nl
    
    echo ""
    echo "Select a workspace by number (or press Enter for default workspace):"
    read -p "Workspace number: " WORKSPACE_SELECTION
    
    if [ -z "$WORKSPACE_SELECTION" ]; then
        # Get default workspace (first one)
        WORKSPACE_ID=$(echo "$WORKSPACES" | jq -r '.workspaces[0].id')
        WORKSPACE_NAME=$(echo "$WORKSPACES" | jq -r '.workspaces[0].name')
    else
        # Get selected workspace
        WORKSPACE_INDEX=$((WORKSPACE_SELECTION - 1))
        WORKSPACE_ID=$(echo "$WORKSPACES" | jq -r ".workspaces[$WORKSPACE_INDEX].id")
        WORKSPACE_NAME=$(echo "$WORKSPACES" | jq -r ".workspaces[$WORKSPACE_INDEX].name")
    fi
    
    if [ "$WORKSPACE_ID" = "null" ] || [ -z "$WORKSPACE_ID" ]; then
        echo "❌ Invalid workspace selection"
        exit 1
    fi
    
    echo "✅ Selected workspace: $WORKSPACE_NAME ($WORKSPACE_ID)"
}

# Upload or update collection
upload_collection() {
    echo ""
    echo "📤 Collection Upload"
    echo "-------------------"
    
    if [ ! -f "postman/GMV_API_Collection.json" ]; then
        echo "❌ Collection file not found: postman/GMV_API_Collection.json"
        echo "Please make sure you're running this from the project root"
        exit 1
    fi
    
    echo "📋 Checking existing collections in workspace..."
    COLLECTIONS=$(curl -s -X GET "https://api.getpostman.com/collections" \
        -H "X-API-Key: $POSTMAN_API_KEY")
    
    COLLECTION_NAME="GMV API Collection"
    EXISTING_COLLECTION_ID=$(echo "$COLLECTIONS" | jq -r ".collections[] | select(.name==\"$COLLECTION_NAME\") | .id")
    
    if [ "$EXISTING_COLLECTION_ID" != "null" ] && [ ! -z "$EXISTING_COLLECTION_ID" ]; then
        echo "🔄 Updating existing collection: $EXISTING_COLLECTION_ID"
        
        # Update existing collection
        curl -s -X PUT "https://api.getpostman.com/collections/$EXISTING_COLLECTION_ID" \
            -H "X-API-Key: $POSTMAN_API_KEY" \
            -H "Content-Type: application/json" \
            -d @postman/GMV_API_Collection.json > /tmp/collection_update.json
        
        COLLECTION_ID=$EXISTING_COLLECTION_ID
    else
        echo "🆕 Creating new collection..."
        
        # Create new collection
        curl -s -X POST "https://api.getpostman.com/collections" \
            -H "X-API-Key: $POSTMAN_API_KEY" \
            -H "Content-Type: application/json" \
            -d @postman/GMV_API_Collection.json > /tmp/collection_create.json
        
        COLLECTION_ID=$(cat /tmp/collection_create.json | jq -r '.collection.id')
        rm /tmp/collection_create.json
    fi
    
    echo "✅ Collection ID: $COLLECTION_ID"
}

# Upload or update environment
upload_environment() {
    echo ""
    echo "🌍 Environment Upload"
    echo "--------------------"
    
    if [ ! -f "postman/GMV_Development_Environment.json" ]; then
        echo "❌ Environment file not found: postman/GMV_Development_Environment.json"
        exit 1
    fi
    
    echo "📋 Checking existing environments..."
    ENVIRONMENTS=$(curl -s -X GET "https://api.getpostman.com/environments" \
        -H "X-API-Key: $POSTMAN_API_KEY")
    
    ENVIRONMENT_NAME="GMV Development Environment"
    EXISTING_ENVIRONMENT_ID=$(echo "$ENVIRONMENTS" | jq -r ".environments[] | select(.name==\"$ENVIRONMENT_NAME\") | .id")
    
    if [ "$EXISTING_ENVIRONMENT_ID" != "null" ] && [ ! -z "$EXISTING_ENVIRONMENT_ID" ]; then
        echo "🔄 Updating existing environment: $EXISTING_ENVIRONMENT_ID"
        
        # Update existing environment
        curl -s -X PUT "https://api.getpostman.com/environments/$EXISTING_ENVIRONMENT_ID" \
            -H "X-API-Key: $POSTMAN_API_KEY" \
            -H "Content-Type: application/json" \
            -d @postman/GMV_Development_Environment.json > /tmp/env_update.json
        
        ENVIRONMENT_ID=$EXISTING_ENVIRONMENT_ID
    else
        echo "🆕 Creating new environment..."
        
        # Create new environment
        curl -s -X POST "https://api.getpostman.com/environments" \
            -H "X-API-Key: $POSTMAN_API_KEY" \
            -H "Content-Type: application/json" \
            -d @postman/GMV_Development_Environment.json > /tmp/env_create.json
        
        ENVIRONMENT_ID=$(cat /tmp/env_create.json | jq -r '.environment.id')
        rm /tmp/env_create.json
    fi
    
    echo "✅ Environment ID: $ENVIRONMENT_ID"
}

# Generate GitHub secrets
generate_github_secrets() {
    echo ""
    echo "🔐 GitHub Secrets Configuration"
    echo "==============================="
    echo ""
    echo "Add these secrets to your GitHub repository:"
    echo "(Go to Settings > Secrets and variables > Actions)"
    echo ""
    echo "POSTMAN_API_KEY = $POSTMAN_API_KEY"
    echo "POSTMAN_WORKSPACE_ID = $WORKSPACE_ID"
    echo "POSTMAN_COLLECTION_ID = $COLLECTION_ID"
    echo "POSTMAN_ENVIRONMENT_ID = $ENVIRONMENT_ID"
    echo ""
    echo "📋 Copy these values to your GitHub repository secrets!"
    echo ""
    
    # Save to file for easy copying
    cat > postman-github-secrets.txt << EOF
# Add these secrets to your GitHub repository
# Go to: Settings > Secrets and variables > Actions

POSTMAN_API_KEY=$POSTMAN_API_KEY
POSTMAN_WORKSPACE_ID=$WORKSPACE_ID  
POSTMAN_COLLECTION_ID=$COLLECTION_ID
POSTMAN_ENVIRONMENT_ID=$ENVIRONMENT_ID
EOF
    
    echo "💾 Secrets saved to: postman-github-secrets.txt"
}

# Show completion summary
show_summary() {
    echo ""
    echo "🎉 Setup Complete!"
    echo "=================="
    echo ""
    echo "✅ What's been configured:"
    echo "   • Postman API connection verified"
    echo "   • Collection uploaded to Postman Cloud"
    echo "   • Environment uploaded to Postman Cloud" 
    echo "   • GitHub secrets generated"
    echo ""
    echo "🔗 Your Postman workspace:"
    echo "   https://go.postman.co/workspace/$WORKSPACE_ID"
    echo ""
    echo "📋 Next steps:"
    echo "   1. Add the generated secrets to your GitHub repository"
    echo "   2. Commit your workflow files to trigger automation"
    echo "   3. Monitor your GitHub Actions for automated testing"
    echo ""
    echo "🚀 Your API testing is now fully automated!"
}

# Main execution flow
main() {
    check_dependencies
    get_postman_api_key
    select_workspace
    upload_collection
    upload_environment
    generate_github_secrets
    show_summary
}

# Run the setup
main