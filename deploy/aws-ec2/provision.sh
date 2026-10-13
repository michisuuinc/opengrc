#!/usr/bin/env bash
# Copyright (c) 2026 MICHISUU Inc.
# SPDX-License-Identifier: MIT
#
# Creates the AWS resources for the in-house environment. Run from a
# workstation with AWS CLI credentials, one stage at a time, in this order
# (see README.md):
#
#   ./provision.sh secrets     # generate secrets into SSM (free)
#   ./provision.sh ses-smtp    # IAM user for SES SMTP + derived password
#   ./provision.sh ecr         # ECR repository (then run build-image.sh)
#   ./provision.sh audit       # session logging, CloudTrail, GuardDuty (asks)
#   ./provision.sh infra       # S3, IAM role, SG, EC2, EIP, snapshots (asks)
#   ./provision.sh ses-domain  # SES domain identity, prints DKIM records
#   ./provision.sh google      # store the Google OAuth client in SSM
#
# Stages are safe to re-run: existing resources and secrets are kept.
# Secrets are generated locally, passed to the AWS CLI through 0600 files
# (never on the command line) and never printed.

set -euo pipefail

REGION="${AWS_REGION:-ap-northeast-1}"
PROJECT="opengrc"
DOMAIN="grc.michisuu.com"
MAIL_DOMAIN="michisuu.com"
SENDER="no-reply@michisuu.com"
SSM_PREFIX="/opengrc"
FILES_BUCKET="michisuu-opengrc-files"
BACKUP_BUCKET="michisuu-opengrc-backup"
TRAIL_BUCKET="michisuu-opengrc-cloudtrail"
ECR_REPO="opengrc/probod"
ROLE_NAME="opengrc-ec2"
SMTP_USER_NAME="opengrc-ses-smtp"
SG_NAME="opengrc-web"
CONTAINER_LOG_GROUP="/opengrc/containers"
SESSION_LOG_GROUP="/opengrc/ssm-sessions"
LOG_RETENTION_DAYS=400
INSTANCE_TYPE="t3.large"
VOLUME_SIZE_GB=40
REPO_URL="https://github.com/michisuuinc/opengrc.git"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export AWS_REGION="$REGION" AWS_PAGER=""

TMP_DIR="$(mktemp -d)"
chmod 700 "$TMP_DIR"
trap 'rm -rf "$TMP_DIR"' EXIT

log() { printf '==> %s\n' "$*" >&2; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

confirm() {
  local answer
  read -r -p "$1 [y/N] " answer
  [[ "$answer" == "y" || "$answer" == "Y" ]] || die "aborted"
}

account_id() { aws sts get-caller-identity --query Account --output text; }

param_exists() {
  aws ssm get-parameter --name "$1" --query Parameter.Name --output text >/dev/null 2>&1
}

# put_secret NAME: stores stdin as a SecureString unless it already exists.
# Existing secrets are never overwritten: replacing the encryption key
# would make stored data unreadable.
put_secret() {
  local name="$SSM_PREFIX/$1" file="$TMP_DIR/value"
  ( umask 077; cat >"$file" )
  if param_exists "$name"; then
    log "keep existing $name"
  else
    aws ssm put-parameter --name "$name" --type SecureString \
      --value "file://$file" --tags "Key=Project,Value=$PROJECT" >/dev/null
    log "stored $name"
  fi
  rm -f "$file"
}

stage_secrets() {
  openssl rand -base64 32 | tr -d '\n' | put_secret encryption-key
  openssl rand -base64 32 | tr -d '\n' | put_secret auth-cookie-secret
  openssl rand -base64 32 | tr -d '\n' | put_secret auth-password-pepper
  openssl rand -hex 24 | tr -d '\n' | put_secret postgres-password
  openssl rand -hex 24 | tr -d '\n' | put_secret probod-pg-password

  if ! param_exists "$SSM_PREFIX/oauth2-signing-key"; then
    ( umask 077; openssl genrsa -out "$TMP_DIR/oauth2.pem" 2048 2>/dev/null )
    put_secret oauth2-signing-key <"$TMP_DIR/oauth2.pem"
  fi

  if ! param_exists "$SSM_PREFIX/saml-private-key"; then
    ( umask 077; openssl req -x509 -newkey rsa:2048 -nodes -days 3650 \
      -subj "/CN=$DOMAIN" -keyout "$TMP_DIR/saml.key" -out "$TMP_DIR/saml.crt" 2>/dev/null )
    put_secret saml-private-key <"$TMP_DIR/saml.key"
    put_secret saml-certificate <"$TMP_DIR/saml.crt"
  fi
  rm -f "$TMP_DIR"/*.pem "$TMP_DIR"/*.key "$TMP_DIR"/*.crt

  cat <<EOF

Keep an offline copy of $SSM_PREFIX/encryption-key (README.md, "暗号化キーの保管").
Without it, data encrypted by probod cannot be recovered if the parameter is lost.
EOF
}

# Reads the create-access-key JSON on stdin and prints "<access key id> <SES
# SMTP password>". The SMTP password is derived from the secret access key
# (AWS "Obtaining Amazon SES SMTP credentials", signature version 4).
ses_smtp_credentials() {
  REGION="$REGION" node -e '
    const { createHmac } = require("node:crypto");
    const key = JSON.parse(require("node:fs").readFileSync(0, "utf8")).AccessKey;
    const sign = (k, msg) => createHmac("sha256", k).update(msg).digest();
    let k = sign("AWS4" + key.SecretAccessKey, "11111111");
    for (const part of [process.env.REGION, "ses", "aws4_request", "SendRawEmail"]) {
      k = sign(k, part);
    }
    const password = Buffer.concat([Buffer.from([0x04]), k]).toString("base64");
    process.stdout.write(key.AccessKeyId + " " + password);
  '
}

stage_ses_smtp() {
  local account
  account="$(account_id)"

  if ! aws iam get-user --user-name "$SMTP_USER_NAME" >/dev/null 2>&1; then
    aws iam create-user --user-name "$SMTP_USER_NAME" \
      --tags "Key=Project,Value=$PROJECT" >/dev/null
    log "created IAM user $SMTP_USER_NAME"
  fi

  # Only raw sending, only through the verified domain identity, only as
  # the configured sender.
  aws iam put-user-policy --user-name "$SMTP_USER_NAME" \
    --policy-name ses-send-raw-email \
    --policy-document "{
      \"Version\": \"2012-10-17\",
      \"Statement\": [{
        \"Effect\": \"Allow\",
        \"Action\": \"ses:SendRawEmail\",
        \"Resource\": \"arn:aws:ses:$REGION:$account:identity/$MAIL_DOMAIN\",
        \"Condition\": {\"StringEquals\": {\"ses:FromAddress\": \"$SENDER\"}}
      }]
    }"

  if param_exists "$SSM_PREFIX/smtp-password"; then
    log "keep existing SMTP credentials (README.md, \"SMTP 認証情報の更新\" to rotate)"
    return
  fi

  local keys
  keys="$(aws iam list-access-keys --user-name "$SMTP_USER_NAME" --query 'length(AccessKeyMetadata)' --output text)"
  [[ "$keys" == "0" ]] || die "$SMTP_USER_NAME already has an access key but SSM has no SMTP password; delete the orphaned key in IAM and re-run"

  aws iam create-access-key --user-name "$SMTP_USER_NAME" --output json \
    | ses_smtp_credentials >"$TMP_DIR/smtp"
  cut -d' ' -f1 <"$TMP_DIR/smtp" | tr -d '\n' | put_secret smtp-user
  cut -d' ' -f2 <"$TMP_DIR/smtp" | tr -d '\n' | put_secret smtp-password
  rm -f "$TMP_DIR/smtp"
}

stage_ecr() {
  if aws ecr describe-repositories --repository-names "$ECR_REPO" >/dev/null 2>&1; then
    log "keep existing ECR repository $ECR_REPO"
  else
    aws ecr create-repository --repository-name "$ECR_REPO" \
      --image-tag-mutability IMMUTABLE \
      --image-scanning-configuration scanOnPush=true \
      --encryption-configuration encryptionType=AES256 \
      --tags "Key=Project,Value=$PROJECT" >/dev/null
    log "created ECR repository $ECR_REPO (immutable tags)"
  fi
  aws ecr put-image-tag-mutability --repository-name "$ECR_REPO" \
    --image-tag-mutability IMMUTABLE >/dev/null

  aws ecr put-lifecycle-policy --repository-name "$ECR_REPO" \
    --lifecycle-policy-text '{
      "rules": [{
        "rulePriority": 1,
        "description": "keep the 10 most recent images",
        "selection": {"tagStatus": "any", "countType": "imageCountMoreThan", "countNumber": 10},
        "action": {"type": "expire"}
      }]
    }' >/dev/null
}

create_log_group() {
  local name="$1"
  if ! aws logs describe-log-groups --log-group-name-prefix "$name" \
      --query "logGroups[?logGroupName=='$name'].logGroupName" --output text | grep -q .; then
    aws logs create-log-group --log-group-name "$name" --tags "Project=$PROJECT"
    log "created log group $name"
  fi
  aws logs put-retention-policy --log-group-name "$name" --retention-in-days "$LOG_RETENTION_DAYS"
}

create_bucket() {
  local bucket="$1"
  if aws s3api head-bucket --bucket "$bucket" >/dev/null 2>&1; then
    log "keep existing bucket $bucket"
  else
    aws s3api create-bucket --bucket "$bucket" \
      --create-bucket-configuration "LocationConstraint=$REGION" >/dev/null
    log "created bucket $bucket"
  fi
  aws s3api put-public-access-block --bucket "$bucket" \
    --public-access-block-configuration \
    BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
  aws s3api put-bucket-encryption --bucket "$bucket" \
    --server-side-encryption-configuration \
    '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}'
  aws s3api put-bucket-versioning --bucket "$bucket" \
    --versioning-configuration Status=Enabled
  aws s3api put-bucket-tagging --bucket "$bucket" \
    --tagging "TagSet=[{Key=Project,Value=$PROJECT}]"
}

# deny_insecure_transport BUCKET [EXTRA_STATEMENT_JSON]
deny_insecure_transport() {
  local bucket="$1" extra="${2:-}"
  aws s3api put-bucket-policy --bucket "$bucket" --policy "{
    \"Version\": \"2012-10-17\",
    \"Statement\": [{
      \"Sid\": \"DenyInsecureTransport\",
      \"Effect\": \"Deny\",
      \"Principal\": \"*\",
      \"Action\": \"s3:*\",
      \"Resource\": [\"arn:aws:s3:::$bucket\", \"arn:aws:s3:::$bucket/*\"],
      \"Condition\": {\"Bool\": {\"aws:SecureTransport\": \"false\"}}
    }${extra:+, $extra}]
  }"
}

setup_buckets() {
  create_bucket "$FILES_BUCKET"
  deny_insecure_transport "$FILES_BUCKET"
  # The console downloads files from presigned S3 URLs in the browser.
  aws s3api put-bucket-cors --bucket "$FILES_BUCKET" --cors-configuration "{
    \"CORSRules\": [{
      \"AllowedOrigins\": [\"https://$DOMAIN\"],
      \"AllowedMethods\": [\"GET\", \"HEAD\"],
      \"AllowedHeaders\": [\"*\"],
      \"MaxAgeSeconds\": 3000
    }]
  }"
  # Deleted files (personal data) must really go away: old versions are
  # kept 30 days for recovery, then removed.
  aws s3api put-bucket-lifecycle-configuration --bucket "$FILES_BUCKET" \
    --lifecycle-configuration '{
      "Rules": [{
        "ID": "expire-old-versions",
        "Status": "Enabled",
        "Filter": {},
        "NoncurrentVersionExpiration": {"NoncurrentDays": 30},
        "AbortIncompleteMultipartUpload": {"DaysAfterInitiation": 7}
      }]
    }'

  create_bucket "$BACKUP_BUCKET"
  deny_insecure_transport "$BACKUP_BUCKET"
  aws s3api put-bucket-lifecycle-configuration --bucket "$BACKUP_BUCKET" \
    --lifecycle-configuration '{
      "Rules": [{
        "ID": "expire-db-dumps",
        "Status": "Enabled",
        "Filter": {"Prefix": "db/"},
        "Expiration": {"Days": 30},
        "NoncurrentVersionExpiration": {"NoncurrentDays": 30},
        "AbortIncompleteMultipartUpload": {"DaysAfterInitiation": 7}
      }]
    }'
}

setup_role() {
  local account
  account="$(account_id)"

  if ! aws iam get-role --role-name "$ROLE_NAME" >/dev/null 2>&1; then
    aws iam create-role --role-name "$ROLE_NAME" \
      --assume-role-policy-document '{
        "Version": "2012-10-17",
        "Statement": [{"Effect": "Allow", "Principal": {"Service": "ec2.amazonaws.com"}, "Action": "sts:AssumeRole"}]
      }' --tags "Key=Project,Value=$PROJECT" >/dev/null
    log "created IAM role $ROLE_NAME"
  fi

  aws iam attach-role-policy --role-name "$ROLE_NAME" \
    --policy-arn arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore

  # Least privilege: the files bucket read/write, the backup bucket write
  # only (a compromised host cannot read or delete backups), one ECR
  # repository, read-only access to its own secrets, and its log groups.
  aws iam put-role-policy --role-name "$ROLE_NAME" --policy-name opengrc-app \
    --policy-document "{
      \"Version\": \"2012-10-17\",
      \"Statement\": [
        {\"Effect\": \"Allow\",
         \"Action\": [\"s3:GetObject\", \"s3:PutObject\", \"s3:DeleteObject\", \"s3:ListBucket\"],
         \"Resource\": [\"arn:aws:s3:::$FILES_BUCKET\", \"arn:aws:s3:::$FILES_BUCKET/*\"]},
        {\"Effect\": \"Allow\",
         \"Action\": \"s3:PutObject\",
         \"Resource\": \"arn:aws:s3:::$BACKUP_BUCKET/db/*\"},
        {\"Effect\": \"Allow\", \"Action\": \"ecr:GetAuthorizationToken\", \"Resource\": \"*\"},
        {\"Effect\": \"Allow\",
         \"Action\": [\"ecr:BatchGetImage\", \"ecr:GetDownloadUrlForLayer\", \"ecr:BatchCheckLayerAvailability\"],
         \"Resource\": \"arn:aws:ecr:$REGION:$account:repository/$ECR_REPO\"},
        {\"Effect\": \"Allow\",
         \"Action\": [\"ssm:GetParameter\", \"ssm:GetParameters\"],
         \"Resource\": \"arn:aws:ssm:$REGION:$account:parameter$SSM_PREFIX/*\"},
        {\"Effect\": \"Deny\",
         \"Action\": [\"ssm:PutParameter\", \"ssm:DeleteParameter\", \"ssm:DeleteParameters\", \"ssm:LabelParameterVersion\"],
         \"Resource\": \"arn:aws:ssm:$REGION:$account:parameter$SSM_PREFIX/*\"},
        {\"Effect\": \"Allow\", \"Action\": \"kms:Decrypt\", \"Resource\": \"*\",
         \"Condition\": {\"StringEquals\": {\"kms:ViaService\": \"ssm.$REGION.amazonaws.com\"}}},
        {\"Effect\": \"Allow\",
         \"Action\": [\"logs:CreateLogStream\", \"logs:PutLogEvents\", \"logs:DescribeLogStreams\"],
         \"Resource\": [\"arn:aws:logs:$REGION:$account:log-group:$CONTAINER_LOG_GROUP:*\",
                        \"arn:aws:logs:$REGION:$account:log-group:$SESSION_LOG_GROUP:*\"]},
        {\"Effect\": \"Allow\", \"Action\": \"logs:DescribeLogGroups\", \"Resource\": \"*\"}
      ]
    }"

  if ! aws iam get-instance-profile --instance-profile-name "$ROLE_NAME" >/dev/null 2>&1; then
    aws iam create-instance-profile --instance-profile-name "$ROLE_NAME" >/dev/null
    aws iam add-role-to-instance-profile --instance-profile-name "$ROLE_NAME" --role-name "$ROLE_NAME"
    log "created instance profile $ROLE_NAME (waiting for IAM propagation)"
    sleep 15
  fi
}

setup_security_group() {
  local vpc sg
  vpc="$(aws ec2 describe-vpcs --filters Name=isDefault,Values=true --query 'Vpcs[0].VpcId' --output text)"
  [[ "$vpc" != "None" ]] || die "no default VPC in $REGION"

  sg="$(aws ec2 describe-security-groups --filters "Name=group-name,Values=$SG_NAME" "Name=vpc-id,Values=$vpc" \
    --query 'SecurityGroups[0].GroupId' --output text)"
  if [[ "$sg" == "None" ]]; then
    sg="$(aws ec2 create-security-group --group-name "$SG_NAME" --vpc-id "$vpc" \
      --description "opengrc: HTTP/HTTPS only" --query GroupId --output text)"
    aws ec2 create-tags --resources "$sg" --tags "Key=Project,Value=$PROJECT"
    for port in 80 443; do
      aws ec2 authorize-security-group-ingress --group-id "$sg" \
        --ip-permissions "IpProtocol=tcp,FromPort=$port,ToPort=$port,IpRanges=[{CidrIp=0.0.0.0/0}],Ipv6Ranges=[{CidrIpv6=::/0}]" >/dev/null
    done
    log "created security group $sg (80/443 only)"
  fi
  check_security_group "$sg"
  echo "$sg"
}

# Fails if the security group allows anything other than TCP 80/443
# (in particular SSH), so drift is caught on every run.
check_security_group() {
  local rules
  rules="$(aws ec2 describe-security-groups --group-ids "$1" \
    --query 'SecurityGroups[0].IpPermissions[].[IpProtocol,FromPort,ToPort]' --output text | sort -u)"
  if [[ "$rules" != $'tcp\t443\t443\ntcp\t80\t80' ]]; then
    printf 'security group %s has unexpected ingress rules:\n%s\n' "$1" "$rules" >&2
    die "remove every rule except TCP 80 and 443"
  fi
}

setup_instance() {
  local sg="$1" instance ami sha user_data
  instance="$(aws ec2 describe-instances \
    --filters "Name=tag:Name,Values=$PROJECT" "Name=instance-state-name,Values=pending,running,stopping,stopped" \
    --query 'Reservations[0].Instances[0].InstanceId' --output text)"
  if [[ "$instance" != "None" ]]; then
    log "keep existing instance $instance"
    echo "$instance"
    return
  fi

  git -C "$SCRIPT_DIR" fetch -q origin main
  sha="$(git -C "$SCRIPT_DIR" rev-parse origin/main)"
  [[ "$sha" =~ ^[0-9a-f]{40}$ ]] || die "cannot resolve origin/main"
  git -C "$SCRIPT_DIR" cat-file -e "$sha:deploy/aws-ec2/user-data.sh" 2>/dev/null \
    || die "deploy/aws-ec2 is not on origin/main yet; merge the PR first"

  ami="$(aws ssm get-parameter \
    --name /aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp3/ami-id \
    --query Parameter.Value --output text)"
  user_data="$(sed -e "s|@REPO_URL@|$REPO_URL|" -e "s|@REPO_SHA@|$sha|" "$SCRIPT_DIR/user-data.sh")"

  instance="$(aws ec2 run-instances \
    --image-id "$ami" --instance-type "$INSTANCE_TYPE" \
    --iam-instance-profile "Name=$ROLE_NAME" \
    --security-group-ids "$sg" \
    --metadata-options HttpTokens=required,HttpPutResponseHopLimit=2,HttpEndpoint=enabled \
    --block-device-mappings "DeviceName=/dev/sda1,Ebs={VolumeSize=$VOLUME_SIZE_GB,VolumeType=gp3,Encrypted=true,DeleteOnTermination=false}" \
    --user-data "$user_data" \
    --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=$PROJECT},{Key=Project,Value=$PROJECT},{Key=Backup,Value=daily}]" \
      "ResourceType=volume,Tags=[{Key=Name,Value=$PROJECT},{Key=Project,Value=$PROJECT},{Key=Backup,Value=daily}]" \
    --query 'Instances[0].InstanceId' --output text)"
  log "launched instance $instance from $sha"
  aws ec2 wait instance-running --instance-ids "$instance"
  echo "$instance"
}

setup_elastic_ip() {
  local instance="$1" alloc
  alloc="$(aws ec2 describe-addresses --filters "Name=tag:Name,Values=$PROJECT" \
    --query 'Addresses[0].AllocationId' --output text)"
  if [[ "$alloc" == "None" ]]; then
    alloc="$(aws ec2 allocate-address --domain vpc \
      --tag-specifications "ResourceType=elastic-ip,Tags=[{Key=Name,Value=$PROJECT},{Key=Project,Value=$PROJECT}]" \
      --query AllocationId --output text)"
    log "allocated Elastic IP $alloc"
  fi
  aws ec2 associate-address --instance-id "$instance" --allocation-id "$alloc" >/dev/null
  aws ec2 describe-addresses --allocation-ids "$alloc" --query 'Addresses[0].PublicIp' --output text
}

setup_snapshots() {
  local account
  account="$(account_id)"
  if aws dlm get-lifecycle-policies --query "Policies[?Description=='$PROJECT daily'].PolicyId" --output text | grep -q .; then
    log "keep existing snapshot policy"
    return
  fi
  if ! aws iam get-role --role-name AWSDataLifecycleManagerDefaultRole >/dev/null 2>&1; then
    aws dlm create-default-role --resource-type snapshot >/dev/null
  fi
  aws dlm create-lifecycle-policy --description "$PROJECT daily" --state ENABLED \
    --execution-role-arn "arn:aws:iam::$account:role/AWSDataLifecycleManagerDefaultRole" \
    --policy-details '{
      "ResourceTypes": ["VOLUME"],
      "TargetTags": [{"Key": "Backup", "Value": "daily"}],
      "Schedules": [{
        "Name": "daily",
        "CreateRule": {"Interval": 24, "IntervalUnit": "HOURS", "Times": ["18:00"]},
        "RetainRule": {"Count": 7},
        "CopyTags": true
      }]
    }' >/dev/null
  log "created daily EBS snapshot policy (03:00 JST, 7 generations)"
}

stage_audit() {
  local account trails detectors
  account="$(account_id)"

  cat <<EOF
This stage changes account-level audit settings in $REGION:
  1. Session Manager preferences (document SSM-SessionManagerRunShell, shared by
     every instance in this account and region): log every session to
     CloudWatch Logs $SESSION_LOG_GROUP (kept $LOG_RETENTION_DAYS days), 20-minute idle timeout.
  2. CloudTrail: creates a multi-region trail with log file validation, only if
     the account has none.
  3. GuardDuty: enables a detector in $REGION, only if none exists.
CloudTrail (S3 storage) and GuardDuty are billed by usage.
EOF
  confirm "Apply these audit settings?"

  create_log_group "$SESSION_LOG_GROUP"
  local prefs
  prefs="{
    \"schemaVersion\": \"1.0\",
    \"description\": \"Session Manager preferences (opengrc)\",
    \"sessionType\": \"Standard_Stream\",
    \"inputs\": {
      \"cloudWatchLogGroupName\": \"$SESSION_LOG_GROUP\",
      \"cloudWatchEncryptionEnabled\": false,
      \"cloudWatchStreamingEnabled\": true,
      \"idleSessionTimeout\": \"20\",
      \"runAsEnabled\": false,
      \"shellProfile\": {\"linux\": \"\", \"windows\": \"\"}
    }
  }"
  printf '%s' "$prefs" >"$TMP_DIR/prefs.json"
  if aws ssm describe-document --name SSM-SessionManagerRunShell >/dev/null 2>&1; then
    aws ssm update-document --name SSM-SessionManagerRunShell \
      --content "file://$TMP_DIR/prefs.json" --document-version '$LATEST' >/dev/null
    aws ssm update-document-default-version --name SSM-SessionManagerRunShell \
      --document-version "$(aws ssm describe-document --name SSM-SessionManagerRunShell --query Document.LatestVersion --output text)" >/dev/null
  else
    aws ssm create-document --name SSM-SessionManagerRunShell --document-type Session \
      --content "file://$TMP_DIR/prefs.json" >/dev/null
  fi
  log "Session Manager sessions are logged to $SESSION_LOG_GROUP"

  trails="$(aws cloudtrail describe-trails --include-shadow-trails --query 'length(trailList)' --output text)"
  if [[ "$trails" == "0" ]]; then
    create_bucket "$TRAIL_BUCKET"
    deny_insecure_transport "$TRAIL_BUCKET" "{
      \"Sid\": \"CloudTrailAclCheck\", \"Effect\": \"Allow\",
      \"Principal\": {\"Service\": \"cloudtrail.amazonaws.com\"},
      \"Action\": \"s3:GetBucketAcl\", \"Resource\": \"arn:aws:s3:::$TRAIL_BUCKET\"
    }, {
      \"Sid\": \"CloudTrailWrite\", \"Effect\": \"Allow\",
      \"Principal\": {\"Service\": \"cloudtrail.amazonaws.com\"},
      \"Action\": \"s3:PutObject\", \"Resource\": \"arn:aws:s3:::$TRAIL_BUCKET/AWSLogs/$account/*\",
      \"Condition\": {\"StringEquals\": {\"s3:x-amz-acl\": \"bucket-owner-full-control\"}}
    }"
    aws cloudtrail create-trail --name "$PROJECT-audit" --s3-bucket-name "$TRAIL_BUCKET" \
      --is-multi-region-trail --enable-log-file-validation >/dev/null
    aws cloudtrail start-logging --name "$PROJECT-audit"
    log "created CloudTrail trail $PROJECT-audit"
  else
    log "keep existing CloudTrail configuration ($trails trail(s))"
  fi

  detectors="$(aws guardduty list-detectors --query 'length(DetectorIds)' --output text)"
  if [[ "$detectors" == "0" ]]; then
    aws guardduty create-detector --enable >/dev/null
    log "enabled GuardDuty in $REGION"
  else
    log "keep existing GuardDuty detector"
  fi
}

stage_infra() {
  param_exists "$SSM_PREFIX/encryption-key" || die "run ./provision.sh secrets first"
  param_exists "$SSM_PREFIX/smtp-password" || die "run ./provision.sh ses-smtp first"
  param_exists "$SSM_PREFIX/image" || die "run build-image.sh first"

  cat <<EOF
This creates billable resources in $REGION (about USD 90-100/month):
  - EC2 $INSTANCE_TYPE (Ubuntu 24.04, no SSH), EBS gp3 ${VOLUME_SIZE_GB}GB encrypted, Elastic IP
  - S3 buckets $FILES_BUCKET and $BACKUP_BUCKET (private, TLS only, versioned)
  - IAM role $ROLE_NAME, security group $SG_NAME (TCP 80/443 only)
  - CloudWatch Logs group $CONTAINER_LOG_GROUP (kept $LOG_RETENTION_DAYS days)
  - Daily EBS snapshots kept for 7 days
EOF
  confirm "Create these resources?"

  setup_buckets
  create_log_group "$CONTAINER_LOG_GROUP"
  setup_role
  local sg instance ip
  sg="$(setup_security_group | tail -n1)"
  instance="$(setup_instance "$sg" | tail -n1)"
  ip="$(setup_elastic_ip "$instance" | tail -n1)"
  setup_snapshots

  cat <<EOF

Instance: $instance
Elastic IP: $ip
Next: add a DNS A record  $DOMAIN -> $ip  (お名前.com), then run
  ./provision.sh ses-domain
EOF
}

stage_ses_domain() {
  if ! aws sesv2 get-email-identity --email-identity "$MAIL_DOMAIN" >/dev/null 2>&1; then
    aws sesv2 create-email-identity --email-identity "$MAIL_DOMAIN" \
      --tags "Key=Project,Value=$PROJECT" >/dev/null
    log "created SES domain identity $MAIL_DOMAIN"
  fi
  echo
  echo "Add these CNAME records to $MAIL_DOMAIN (お名前.com):"
  aws sesv2 get-email-identity --email-identity "$MAIL_DOMAIN" \
    --query 'DkimAttributes.Tokens' --output text | tr '\t' '\n' | while read -r token; do
      printf '  %s._domainkey.%s  CNAME  %s.dkim.amazonses.com\n' "$token" "$MAIL_DOMAIN" "$token"
    done
  echo
  aws sesv2 get-email-identity --email-identity "$MAIL_DOMAIN" \
    --query '{verified: VerifiedForSendingStatus, dkim: DkimAttributes.Status}' --output table
}

stage_google() {
  local client_id client_secret
  read -r -p "Google OAuth client ID: " client_id
  read -r -s -p "Google OAuth client secret: " client_secret
  echo
  [[ -n "$client_id" && -n "$client_secret" ]] || die "both values are required"
  printf '%s' "$client_id" | put_secret google-client-id
  printf '%s' "$client_secret" | put_secret google-client-secret
  echo "Now uncomment the PROBOD_AUTH_GOOGLE_* lines in probod-settings.conf, merge, and run deploy.sh."
}

case "${1:-}" in
  secrets) stage_secrets ;;
  ses-smtp) stage_ses_smtp ;;
  ecr) stage_ecr ;;
  audit) stage_audit ;;
  infra) stage_infra ;;
  ses-domain) stage_ses_domain ;;
  google) stage_google ;;
  *) sed -n '5,19p' "$0"; exit 2 ;;
esac
