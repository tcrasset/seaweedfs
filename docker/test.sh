#!/usr/bin/env bash

set -euo pipefail

export PAGER=cat

BUCKET_NAME="test-bucket-$(openssl rand -hex 4)"
SENTINEL_FILE=/tmp/SENTINEL
S3_URL="http://127.0.0.1:8333"
IAM_URL="http://127.0.0.1:8111"


function get_user_dir {
    local user="$1"
    echo "$BUCKET_NAME/user-id-$user/"
}

echo "Creating Power User"
power_user_key="power_user_key"
power_user_secret="power_user_secret"
echo "s3.configure -apply -user poweruser -access_key $power_user_key -secret_key $power_user_secret -actions Admin" | docker exec -i seaweedfs-master-1 weed shell > /dev/null

echo "Power User created with key: $power_user_key and secret: $power_user_secret"

export AWS_ACCESS_KEY_ID=$power_user_key
export AWS_SECRET_ACCESS_KEY=$power_user_secret
export AWS_DEFAULT_REGION=us-east-1


echo "Creating Bucket $BUCKET_NAME"
aws --endpoint-url "$S3_URL" s3 mb "s3://$BUCKET_NAME"

echo "Creating SENTINEL file"
echo "Hello World" > "$SENTINEL_FILE"

function upload_sentinel {
    local user="$1"
    echo "Uploading SENTINEL file for $user"
    aws --endpoint-url "$S3_URL" s3 cp "$SENTINEL_FILE" "s3://$(get_user_dir $user)"
}

function upload_file {
    local user="$1"
    local file="$2"
    echo "Uploading $file for $user"
    aws --endpoint-url "$S3_URL" s3 cp "$file" "s3://$(get_user_dir $user)"
}

function create_user {
    local user="$1"
    output="$(aws --endpoint $IAM_URL iam create-access-key --user-name $user)"
    echo "$output"
}

function list_files {    
    local access_key="$1"
    local secret="$2"
    local path="s3://$BUCKET_NAME/"
    
    if [ -n "${3-}" ]; then
        path="$3"
    fi

    echo "Listing files of $path as $access_key"
    AWS_ACCESS_KEY_ID="$access_key" AWS_SECRET_ACCESS_KEY="$secret" aws --endpoint-url "$S3_URL" s3 ls "$path" 2>&1 || true
}



function create_read_only_policy_for_user {
    local user="$1"
    local bucket="$2"

    echo "Creating read only policy for $user on $bucket"

    echo "
    {
        \"Version\": \"2012-10-17\",
        \"Statement\": [
            {
                \"Effect\": \"Allow\",
                \"Action\": [
                    \"s3:Get*\",
                    \"s3:List*\",
                    \"s3:Put*\",
                    \"s3:Delete*\"
                ],
                \"Resource\": [
                    \"arn:aws:s3:::$(get_user_dir $user)\",
                    \"arn:aws:s3:::$(get_user_dir $user)*\"
                ]
            }
        ]
    }
    " > "/tmp/$user-$bucket-read-only.policy"
    cat "/tmp/$user-$bucket-read-only.policy"
    aws --endpoint "$IAM_URL" iam create-policy --policy-name "$user-$bucket-read-only" --policy-document file:///tmp/$user-$bucket-read-only.policy
}

upload_sentinel "Alice"

upload_sentinel "Bob"

list_files "$power_user_key" "$power_user_secret"

echo "Creating user Alice"
alice_user_info=$(create_user "Alice")
echo "Creating user Bob"
bob_user_info=$(create_user "Bob")

function get_user_key {
    local user_info="$1"

    echo "$user_info" | jq -r '.AccessKey.AccessKeyId'
}

function get_user_secret {
    local user_info="$1"
    echo "$user_info" | jq -r '.AccessKey.SecretAccessKey'
}


alice_key="$(get_user_key "$alice_user_info")"
alice_secret="$(get_user_secret "$alice_user_info")"

bob_key="$(get_user_key "$bob_user_info")"
bob_secret="$(get_user_secret "$bob_user_info")"



echo "############"
echo "### Making sure Admin can still list files"
echo "############"
list_files "$power_user_key" "$power_user_secret"

echo "############"
echo "### Making sure Alice can't read any files yet"
echo "############"
list_files "$alice_key" "$alice_secret"
echo "############"
echo "### Making sure Bob can't read any files yet"
echo "############"
list_files "$bob_key" "$bob_secret"


create_read_only_policy_for_user "Alice" $BUCKET_NAME
create_read_only_policy_for_user "Bob" $BUCKET_NAME


echo "############"
echo "### Uploading file as Alice"
echo "############"
AWS_ACCESS_KEY_ID="$alice_key" AWS_SECRET_ACCESS_KEY="$alice_secret" upload_file "Alice" "$SENTINEL_FILE"

echo "############"
echo "#### Making sure that Alice can't read any except their own files"
echo "############"


list_files "$alice_key" "$alice_secret"
list_files "$alice_key" "$alice_secret" "s3://$(get_user_dir Bob)"
list_files "$alice_key" "$alice_secret" "s3://$(get_user_dir Alice)"





