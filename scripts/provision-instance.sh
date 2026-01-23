#!/bin/bash

# VideoStreamGo Instance Provisioning Script
# This script provisions a new tenant instance including:
# - Database creation and migration
# - S3 bucket creation with IAM policies
# - DNS configuration for subdomain
# - SSL certificate provisioning (Let's Encrypt)
# - Initial configuration seeding
# - Health verification

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/videostreamgo/provision-$(date +%Y%m%d-%H%M%S).log"
CONFIG_FILE="${SCRIPT_DIR}/../.env"

# Default values
INSTANCE_ID=""
INSTANCE_NAME=""
SUBDOMAIN=""
DATABASE_NAME=""
DATABASE_USER=""
DATABASE_PASSWORD=""
STORAGE_BUCKET=""
MASTER_DB_HOST="localhost"
MASTER_DB_PORT="5432"
MASTER_DB_USER="videostreamgo"
MASTER_DB_PASSWORD=""
S3_ENDPOINT=""
S3_ACCESS_KEY=""
S3_SECRET_KEY=""
DNS_PROVIDER="cloudflare"
DNS_API_TOKEN=""
LETSENCRYPT_EMAIL=""
PLATFORM_DOMAIN="videostreamgo.com"

# Functions
log() {
    local level=$1
    local message=$2
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${timestamp} [${level}] ${message}" | tee -a "$LOG_FILE"
}

info() {
    log "INFO" "${BLUE}${1}${NC}"
}

success() {
    log "SUCCESS" "${GREEN}${1}${NC}"
}

warn() {
    log "WARN" "${YELLOW}${1}${NC}"
}

error() {
    log "ERROR" "${RED}${1}${NC}"
}

header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  ${1}${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
}

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -i, --instance-id UUID      Instance ID (UUID format)"
    echo "  -n, --instance-name NAME    Instance name"
    echo "  -s, --subdomain SUBDOMAIN   Subdomain for the instance"
    echo "  -d, --database-name NAME    Database name (auto-generated if not provided)"
    echo "  -u, --database-user USER    Database user (auto-generated if not provided)"
    echo "  -p, --database-password PASS Database password (auto-generated if not provided)"
    echo "  -b, --storage-bucket BUCKET S3 bucket name (auto-generated if not provided)"
    echo "  -h, --help                  Show this help message"
    echo ""
    echo "Environment variables:"
    echo "  MASTER_DB_HOST              Master database host (default: localhost)"
    echo "  MASTER_DB_PORT              Master database port (default: 5432)"
    echo "  MASTER_DB_USER              Master database user (default: videostreamgo)"
    echo "  MASTER_DB_PASSWORD          Master database password"
    echo "  S3_ENDPOINT                 S3 endpoint URL"
    echo "  S3_ACCESS_KEY               S3 access key"
    echo "  S3_SECRET_KEY               S3 secret key"
    echo "  DNS_API_TOKEN               DNS provider API token"
    echo "  LETSENCRYPT_EMAIL           Email for Let's Encrypt certificates"
    echo ""
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -i|--instance-id)
                INSTANCE_ID="$2"
                shift 2
                ;;
            -n|--instance-name)
                INSTANCE_NAME="$2"
                shift 2
                ;;
            -s|--subdomain)
                SUBDOMAIN="$2"
                shift 2
                ;;
            -d|--database-name)
                DATABASE_NAME="$2"
                shift 2
                ;;
            -u|--database-user)
                DATABASE_USER="$2"
                shift 2
                ;;
            -p|--database-password)
                DATABASE_PASSWORD="$2"
                shift 2
                ;;
            -b|--storage-bucket)
                STORAGE_BUCKET="$2"
                shift 2
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Load configuration from .env file
load_config() {
    if [[ -f "$CONFIG_FILE" ]]; then
        info "Loading configuration from $CONFIG_FILE"
        set -a
        source "$CONFIG_FILE"
        set +a
    fi

    # Override with environment variables if set
    MASTER_DB_HOST="${MASTER_DB_HOST:-localhost}"
    MASTER_DB_PORT="${MASTER_DB_PORT:-5432}"
    MASTER_DB_USER="${MASTER_DB_USER:-videostreamgo}"
}

# Validate required parameters
validate_params() {
    if [[ -z "$INSTANCE_ID" ]]; then
        error "Instance ID is required. Use -i or --instance-id"
        exit 1
    fi

    if [[ -z "$SUBDOMAIN" ]]; then
        error "Subdomain is required. Use -s or --subdomain"
        exit 1
    fi

    # Validate UUID format
    if ! [[ "$INSTANCE_ID" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]]; then
        error "Invalid UUID format for instance ID: $INSTANCE_ID"
        exit 1
    fi

    # Validate subdomain format
    if ! [[ "$SUBDOMAIN" =~ ^[a-z0-9][a-z0-9-]*[a-z0-9]$ ]]; then
        error "Invalid subdomain format: $SUBDOMAIN"
        exit 1
    fi

    success "Parameters validated successfully"
}

# Generate secure passwords
generate_password() {
    openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32
}

# Provision database
provision_database() {
    header "Provisioning Database"

    # Generate database credentials if not provided
    if [[ -z "$DATABASE_NAME" ]]; then
        DATABASE_NAME="instance_${INSTANCE_ID:0:8}"
    fi

    if [[ -z "$DATABASE_USER" ]]; then
        DATABASE_USER="instance_${INSTANCE_ID:0:8}"
    fi

    if [[ -z "$DATABASE_PASSWORD" ]]; then
        DATABASE_PASSWORD=$(generate_password)
    fi

    info "Creating database: $DATABASE_NAME"
    info "Creating database user: $DATABASE_USER"

    # Create database and user
    PGPASSWORD="$MASTER_DB_PASSWORD" psql -h "$MASTER_DB_HOST" -p "$MASTER_DB_PORT" -U "$MASTER_DB_USER" << EOF
-- Create database
CREATE DATABASE "$DATABASE_NAME" WITH OWNER "$MASTER_DB_USER";

-- Create database user
DO \$\$BEGIN
    CREATE USER "$DATABASE_USER" WITH PASSWORD '$DATABASE_PASSWORD';
EXCEPTION
    WHEN duplicate_object THEN null;
END\$\$;

-- Grant privileges
ALTER DATABASE "$DATABASE_NAME" OWNER TO "$DATABASE_USER";
ALTER USER "$DATABASE_USER" WITH PASSWORD '$DATABASE_PASSWORD';

-- Grant access to database
GRANT ALL PRIVILEGES ON DATABASE "$DATABASE_NAME" TO "$DATABASE_USER";

-- Connect to database and grant schema permissions
\c "$DATABASE_NAME"
GRANT ALL ON SCHEMA public TO "$DATABASE_USER";
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO "$DATABASE_USER";
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO "$DATABASE_USER";
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO "$DATABASE_USER";
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO "$DATABASE_USER";
EOF

    if [[ $? -eq 0 ]]; then
        success "Database created successfully"
    else
        error "Failed to create database"
        exit 1
    fi

    # Run migrations
    info "Running database migrations"
    PGPASSWORD="$MASTER_DB_PASSWORD" psql -h "$MASTER_DB_HOST" -p "$MASTER_DB_PORT" -U "$MASTER_DB_USER" -d "$DATABASE_NAME" -f "${SCRIPT_DIR}/../sql/instance_schema.sql" 2>&1 | tee -a "$LOG_FILE"

    if [[ $? -eq 0 ]]; then
        success "Database migrations completed"
    else
        warn "Migration file not found, skipping"
    fi

    # Store credentials securely
    save_credentials
}

# Save credentials to file
save_credentials() {
    local creds_file="${SCRIPT_DIR}/../credentials/${INSTANCE_ID}.conf"
    mkdir -p "$(dirname "$creds_file")"

    cat > "$creds_file" << EOF
# Instance Credentials - $INSTANCE_ID
# Generated on $(date)

DATABASE_NAME=$DATABASE_NAME
DATABASE_USER=$DATABASE_USER
DATABASE_PASSWORD=$DATABASE_PASSWORD
STORAGE_BUCKET=$STORAGE_BUCKET

# Store securely in production!
EOF

    chmod 600 "$creds_file"
    info "Credentials saved to $creds_file"
}

# Provision S3 bucket
provision_storage() {
    header "Provisioning Storage"

    if [[ -z "$STORAGE_BUCKET" ]]; then
        STORAGE_BUCKET="instance-${INSTANCE_ID:0:8}"
    fi

    info "Creating S3 bucket: $STORAGE_BUCKET"

    # Use AWS CLI if available, otherwise use mc (MinIO Client)
    if command -v aws &> /dev/null; then
        aws s3 mb "s3://${STORAGE_BUCKET}" 2>&1 | tee -a "$LOG_FILE" || true
        aws s3api put-bucket-versioning \
            --bucket "$STORAGE_BUCKET" \
            --versioning-configuration Status=Enabled 2>&1 | tee -a "$LOG_FILE"

        # Create IAM policy for this bucket
        create_iam_policy
    elif command -v mc &> /dev/null; then
        mc mb "s3/${STORAGE_BUCKET}" 2>&1 | tee -a "$LOG_FILE" || true
    else
        warn "Neither AWS CLI nor MinIO Client found. Skipping S3 bucket creation."
        warn "Please create the bucket manually."
    fi

    success "Storage bucket configured"
}

# Create IAM policy for tenant
create_iam_policy() {
    info "Creating IAM policy for instance"

    local policy_name="videostreamgo-instance-${INSTANCE_ID:0:8}"
    local policy_arn=$(aws iam create-policy \
        --policy-name "$policy_name" \
        --policy-document "{
            \"Version\": \"2012-10-17\",
            \"Statement\": [
                {
                    \"Effect\": \"Allow\",
                    \"Action\": [
                        \"s3:PutObject\",
                        \"s3:GetObject\",
                        \"s3:DeleteObject\"
                    ],
                    \"Resource\": \"arn:aws:s3:::${STORAGE_BUCKET}/*\"
                },
                {
                    \"Effect\": \"Allow\",
                    \"Action\": [
                        \"s3:ListBucket\"
                    ],
                    \"Resource\": \"arn:aws:s3:::${STORAGE_BUCKET}\"
                }
            ]
        }" \
        --query 'Policy.Arn' \
        --output text 2>/dev/null)

    if [[ -n "$policy_arn" ]]; then
        success "IAM policy created: $policy_arn"
    else
        warn "Failed to create IAM policy or policy already exists"
    fi
}

# Configure DNS
configure_dns() {
    header "Configuring DNS"

    local full_domain="${SUBDOMAIN}.${PLATFORM_DOMAIN}"

    info "Configuring DNS for: $full_domain"

    case "$DNS_PROVIDER" in
        cloudflare)
            configure_cloudflare_dns
            ;;
        route53)
            configure_route53_dns
            ;;
        *)
            warn "Unknown DNS provider: $DNS_PROVIDER"
            warn "Please configure DNS manually: $full_domain -> $(get_public_ip)"
            ;;
    esac

    success "DNS configuration completed"
}

# Configure Cloudflare DNS
configure_cloudflare_dns() {
    if [[ -z "$DNS_API_TOKEN" ]]; then
        warn "DNS_API_TOKEN not set. Skipping Cloudflare DNS configuration."
        return
    fi

    # Get Cloudflare zone ID
    local zone_id=$(curl -s -X GET \
        "https://api.cloudflare.com/client/v4/zones?name=${PLATFORM_DOMAIN}" \
        -H "Authorization: Bearer ${DNS_API_TOKEN}" \
        -H "Content-Type: application/json" \
        | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [[ -z "$zone_id" ]]; then
        warn "Failed to get Cloudflare zone ID"
        return
    fi

    # Get public IP
    local public_ip=$(get_public_ip)

    # Create A record
    info "Creating A record for ${SUBDOMAIN}.${PLATFORM_DOMAIN} -> $public_ip"

    curl -s -X POST \
        "https://api.cloudflare.com/client/v4/zones/${zone_id}/dns_records" \
        -H "Authorization: Bearer ${DNS_API_TOKEN}" \
        -H "Content-Type: application/json" \
        --data "{
            \"type\": \"A\",
            \"name\": \"${SUBDOMAIN}\",
            \"content\": \"${public_ip}\",
            \"ttl\": 3600,
            \"proxied\": false
        }" | tee -a "$LOG_FILE"

    success "Cloudflare DNS configured"
}

# Configure Route53 DNS
configure_route53_dns() {
    info "Route53 DNS configuration would be added here"
    warn "Please configure Route53 DNS manually"
}

# Get public IP address
get_public_ip() {
    curl -s ifconfig.me || curl -s ipinfo.io/ip || echo "203.0.113.1"
}

# Provision SSL certificate
provision_ssl() {
    header "Provisioning SSL Certificate"

    local domain="${SUBDOMAIN}.${PLATFORM_DOMAIN}"

    if [[ -z "$LETSENCRYPT_EMAIL" ]]; then
        warn "LETSENCRYPT_EMAIL not set. Skipping SSL certificate provisioning."
        warn "Please provision SSL certificate manually for: $domain"
        return
    fi

    info "Requesting SSL certificate for: $domain"

    # Check if certbot is available
    if command -v certbot &> /dev/null; then
        certbot certonly \
            --non-interactive \
            --email "$LETSENCRYPT_EMAIL" \
            --agree-tos \
            --dns-cloudflare \
            --dns-cloudflare-credentials "${SCRIPT_DIR}/../cloudflare.ini" \
            -d "$domain" \
            -d "*.${domain}" 2>&1 | tee -a "$LOG_FILE"

        if [[ $? -eq 0 ]]; then
            success "SSL certificate provisioned"
            info "Certificate location: /etc/letsencrypt/live/${domain}/"
        else
            warn "SSL certificate provisioning failed"
        fi
    else
        warn "certbot not found. Skipping SSL certificate provisioning."
    fi
}

# Seed initial configuration
seed_config() {
    header "Seeding Initial Configuration"

    info "Creating instance configuration record"

    # Insert instance configuration into master database
    PGPASSWORD="$MASTER_DB_PASSWORD" psql -h "$MASTER_DB_HOST" -p "$MASTER_DB_PORT" -U "$MASTER_DB_USER" << EOF
INSERT INTO instance_config (instance_id, config_key, config_value, created_at, updated_at)
VALUES
    ('$INSTANCE_ID', 'site_name', '$INSTANCE_NAME', NOW(), NOW()),
    ('$INSTANCE_ID', 'subdomain', '$SUBDOMAIN', NOW(), NOW()),
    ('$INSTANCE_ID', 'database_name', '$DATABASE_NAME', NOW(), NOW()),
    ('$INSTANCE_ID', 'storage_bucket', '$STORAGE_BUCKET', NOW(), NOW()),
    ('$INSTANCE_ID', 'max_storage_gb', '100', NOW(), NOW()),
    ('$INSTANCE_ID', 'max_videos', '10000', NOW(), NOW()),
    ('$INSTANCE_ID', 'max_users', '10000', NOW(), NOW()),
    ('$INSTANCE_ID', 'primary_color', '#2563eb', NOW(), NOW()),
    ('$INSTANCE_ID', 'video_upload_enabled', 'true', NOW(), NOW()),
    ('$INSTANCE_ID', 'user_registration_enabled', 'true', NOW(), NOW())
ON CONFLICT (instance_id, config_key) DO UPDATE SET
    config_value = EXCLUDED.config_value,
    updated_at = NOW();
EOF

    if [[ $? -eq 0 ]]; then
        success "Configuration seeded successfully"
    else
        error "Failed to seed configuration"
    fi
}

# Verify health
verify_health() {
    header "Verifying Instance Health"

    local full_domain="${SUBDOMAIN}.${PLATFORM_DOMAIN}"

    # Check database connectivity
    info "Checking database connectivity..."
    if PGPASSWORD="$MASTER_DB_PASSWORD" psql -h "$MASTER_DB_HOST" -p "$MASTER_DB_PORT" -U "$DATABASE_USER" -d "$DATABASE_NAME" -c "SELECT 1" &> /dev/null; then
        success "Database connection: OK"
    else
        error "Database connection: FAILED"
    fi

    # Check storage bucket
    info "Checking storage bucket..."
    if command -v aws &> /dev/null; then
        if aws s3 ls "s3://${STORAGE_BUCKET}/" &> /dev/null; then
            success "Storage bucket: OK"
        else
            warn "Storage bucket: Not accessible (may need IAM role)"
        fi
    fi

    # Check DNS resolution
    info "Checking DNS resolution..."
    if host "$full_domain" &> /dev/null; then
        success "DNS resolution: OK ($full_domain)"
    else
        warn "DNS resolution: Pending (may need time to propagate)"
    fi

    # Update instance status
    info "Updating instance status in master database..."

    PGPASSWORD="$MASTER_DB_PASSWORD" psql -h "$MASTER_DB_HOST" -p "$MASTER_DB_PORT" -U "$MASTER_DB_USER" -d "videostreamgo_master" << EOF
UPDATE instances
SET status = 'active',
    database_name = '$DATABASE_NAME',
    storage_bucket = '$STORAGE_BUCKET',
    activated_at = NOW(),
    updated_at = NOW()
WHERE id = '$INSTANCE_ID';
EOF

    if [[ $? -eq 0 ]]; then
        success "Instance status updated to: active"
    else
        error "Failed to update instance status"
    fi

    header "Provisioning Complete!"
    echo ""
    echo -e "${GREEN}Instance has been successfully provisioned:${NC}"
    echo ""
    echo "  Instance ID:     $INSTANCE_ID"
    echo "  Instance Name:   $INSTANCE_NAME"
    echo "  URL:             https://${full_domain}"
    echo "  Database:        $DATABASE_NAME"
    echo "  Storage Bucket:  $STORAGE_BUCKET"
    echo ""
    echo -e "${YELLOW}Next steps:${NC}"
    echo "  1. Verify DNS propagation"
    echo "  2. Complete SSL certificate provisioning"
    echo "  3. Configure custom domain (if needed)"
    echo "  4. Set up monitoring"
    echo ""
}

# Main execution
main() {
    header "VideoStreamGo Instance Provisioning Script"

    # Initialize log file
    mkdir -p "$(dirname "$LOG_FILE")"
    touch "$LOG_FILE"

    # Parse arguments
    parse_args "$@"

    # Load configuration
    load_config

    # Validate parameters
    validate_params

    # Generate instance name if not provided
    if [[ -z "$INSTANCE_NAME" ]]; then
        INSTANCE_NAME="${SUBDOMAIN^}"
    fi

    info "Starting provisioning for instance: $INSTANCE_ID"
    info "Subdomain: $SUBDOMAIN"
    info "Log file: $LOG_FILE"

    # Provision resources
    provision_database
    provision_storage
    configure_dns
    provision_ssl
    seed_config
    verify_health

    success "Provisioning completed successfully!"
}

# Run main function
main "$@"
