#!/bin/bash
set -e

ENVIRONMENT=${1:-dev}
AWS_REGION=${2:-ap-southeast-2}
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

echo "Building Lambda layer for cryptography..."

# Create temp directory
TEMP_DIR=$(mktemp -d)
LAYER_DIR="$TEMP_DIR/python"
mkdir -p "$LAYER_DIR"

# Install cryptography into the layer directory
echo "Installing cryptography library..."
pip install cryptography -t "$LAYER_DIR" --platform manylinux2014_x86_64 --only-binary=:all: --python-version 3.11

# Create zip file using Python
echo "Creating layer zip file..."
cd "$TEMP_DIR"
python3 -c "
import zipfile
import os
from pathlib import Path

zip_path = 'cryptography-layer.zip'
with zipfile.ZipFile(zip_path, 'w', zipfile.ZIP_DEFLATED) as zipf:
    for root, dirs, files in os.walk('python'):
        for file in files:
            file_path = os.path.join(root, file)
            arcname = file_path
            zipf.write(file_path, arcname)
print(f'Created {zip_path}')
"

# Create S3 bucket if it doesn't exist
BUCKET_NAME="cloudfront-key-lambda-layers-${ACCOUNT_ID}-${ENVIRONMENT}"
echo "Checking S3 bucket: $BUCKET_NAME"

if ! aws s3 ls "s3://$BUCKET_NAME" 2>/dev/null; then
    echo "Creating S3 bucket..."
    if [ "$AWS_REGION" == "us-east-1" ]; then
        aws s3 mb "s3://$BUCKET_NAME" --region "$AWS_REGION"
    else
        aws s3api create-bucket \
            --bucket "$BUCKET_NAME" \
            --region "$AWS_REGION" \
            --create-bucket-configuration LocationConstraint="$AWS_REGION"
    fi
fi

# Upload to S3
echo "Uploading layer to S3..."
aws s3 cp cryptography-layer.zip "s3://$BUCKET_NAME/cryptography-layer.zip"

# Cleanup
rm -rf "$TEMP_DIR"

echo "Layer built and uploaded successfully!"
echo "Bucket: $BUCKET_NAME"
echo "Key: cryptography-layer.zip"
