# AWS RDS to Deeb Onboarding Scripts

This directory contains Python scripts to automate the onboarding of AWS RDS instances and Aurora clusters into Deeb.

## Overview

The onboarding process consists of two phases:

1. **Discovery Phase**: Scan your AWS account for RDS instances and Aurora clusters, generating a YAML configuration file for review
2. **Onboarding Phase**: Process the reviewed configuration to create Deeb users and register instances in Deeb

## Prerequisites

### Python Dependencies

Install the required Python packages:

```bash
pip3 install -r ../requirements-aws.txt
```

### AWS Configuration

- AWS credentials configured (via `~/.aws/credentials` or environment variables)
- Appropriate IAM permissions to:
  - List RDS instances and clusters (`rds:DescribeDBInstances`, `rds:DescribeDBClusters`)
  - List resource tags (`rds:ListTagsForResource`)
  - Retrieve secrets from Secrets Manager (`secretsmanager:GetSecretValue`)

### Deeb Configuration

- Access to a running Deeb instance
- API authentication token
- Project ID where database instances will be created
- Environments configured in Deeb that match your RDS Environment tags

### RDS Requirements

- RDS instances/clusters must have master credentials managed by AWS Secrets Manager
- RDS instances must be tagged with an `Environment` tag (e.g., `production`, `staging`, `development`)
- **Network access from your machine to the RDS instances** - This is critical:
  - RDS instances must be accessible from your workstation
  - You may need to connect via VPN
  - RDS security groups must allow your IP address
  - Test connectivity: `nc -zv <rds-endpoint> 3306` (MySQL) or `nc -zv <rds-endpoint> 5432` (PostgreSQL)

## Workflow

### Phase 1: Discovery

Run the discovery script to scan your AWS account:

```bash
cd scripts/aws-onboarding

# Discover ALL RDS instances/clusters (default)
python3 aws-discover-rds.py

# Or filter by specific environment
python3 aws-discover-rds.py --environment production
python3 aws-discover-rds.py --environment staging
python3 aws-discover-rds.py -e homol

# Specify custom output file
python3 aws-discover-rds.py --environment production --output prod-only.yaml

# Enable verbose logging
python3 aws-discover-rds.py --verbose
```

**What it does:**
- Scans all RDS instances and Aurora clusters in your AWS region (or filters by Environment tag)
- Extracts the `Environment` tag from each resource
- Groups instances by environment
- Generates `rds-discovered.yaml` with the discovered resources
- Logs instances without Secrets Manager to `rds-no-secrets-manager.log`

**Command Options:**
- `--environment, -e ENV`: Filter by Environment tag (case-insensitive). Default: discover all environments
- `--output, -o FILE`: Output YAML file path (default: rds-discovered.yaml)
- `--verbose, -v`: Enable verbose/debug logging

**Output files:**
- `rds-discovered.yaml`: Configuration file with all discovered instances
- `rds-no-secrets-manager.log`: List of instances that don't use Secrets Manager (these will be skipped)

**Review the YAML file:**

```yaml
environments:
  production:
    - name: "prod-mysql-01"
      engine: "mysql"
      endpoint: "prod-mysql-01.xxx.rds.amazonaws.com"
      port: 3306
      master_secret_arn: "arn:aws:secretsmanager:..."
      is_cluster: false
      db_instance_identifier: "prod-mysql-01"
      arn: "arn:aws:rds:..."
  staging:
    - name: "staging-postgres-01"
      engine: "postgresql"
      ...
```

**Important:** Review this file carefully and remove any instances you don't want to onboard to Deeb.

### Phase 2: Configuration

Create your Deeb configuration file:

```bash
cp config.yaml.example config.yaml
```

Edit `config.yaml` with your settings:

```yaml
deeb_api_url: "https://your-deeb-instance.com"
deeb_api_token: "your-api-token"
deeb_project_id: "your-project-uuid"
aws_region: "us-east-1"
```

**How to obtain these values:**

- **deeb_api_url**: The URL of your Deeb instance
- **deeb_api_token**: Generate from your Deeb user settings or API keys section
- **deeb_project_id**: Find in Deeb UI under Projects (UUID format)
- **aws_region**: The AWS region where your RDS instances are located

### Phase 3: Onboarding

Run the onboarding script:

```bash
python3 aws-onboard-to-deeb.py rds-discovered.yaml
```

Or specify a custom config file:

```bash
python3 aws-onboard-to-deeb.py rds-discovered.yaml my-config.yaml
```

**What it does:**

For each RDS instance in the YAML file:

1. **Retrieve master credentials** from AWS Secrets Manager
2. **Create Deeb user** on the RDS instance:
   - For MySQL: Creates user `deeb@'%'` with production-safe privileges
   - For PostgreSQL: Creates user `deeb` with RDS IAM, CREATEDB, and owner role grants
3. **Generate secure password** (32 characters, random)
4. **Match environment** to Deeb environment (case-insensitive)
5. **Register instance** in Deeb via API

**Output logs:**
- Console output with progress
- `rds-onboarding-success.log`: Successfully onboarded instances
- `rds-onboarding-failed.log`: Failed instances with error messages

## SQL Scripts Reference

The scripts use the following SQL templates:

### MySQL (`.local/create-deeb-user-mysql.sql`)

Creates a Deeb user with production-safe privileges:
- ALTER, CREATE, INSERT, UPDATE, SELECT, etc.
- Excludes DROP, DELETE, TRUNCATE for safety

### PostgreSQL (`.local/create-deeb-user-pg.sql`)

Creates a Deeb user with:
- RDS IAM authentication support
- CREATEDB privilege
- Automatic grants of database owner roles for access to all databases

## Troubleshooting

### Instance not using Secrets Manager

**Error**: Instance listed in `rds-no-secrets-manager.log`

**Solution**: Enable Secrets Manager for the instance:
1. In AWS Console, go to RDS
2. Select the instance
3. Modify → Manage master credentials in Secrets Manager
4. Re-run the discovery script

### DNS resolution failure

**Error**: `Can't connect to MySQL server... nodename nor servname provided, or not known`

**Solution**:
This means the RDS endpoint cannot be resolved via DNS, typically due to network connectivity:

1. **Verify you're connected to VPN** if RDS is in a private subnet
2. **Check network connectivity**:
   ```bash
   # Test DNS resolution
   nslookup <rds-endpoint>
   
   # Test port connectivity
   nc -zv <rds-endpoint> 3306  # MySQL
   nc -zv <rds-endpoint> 5432  # PostgreSQL
   ```
3. **Verify RDS security groups** allow inbound traffic from your IP:
   - Go to RDS Console → Select instance → Security groups
   - Ensure inbound rules allow MySQL (3306) or PostgreSQL (5432) from your IP
4. **Check if RDS is publicly accessible**:
   - Private RDS instances require VPN or bastion host access
   - Public RDS instances need proper security group rules

### Connection timeout

**Error**: `Failed to create [MySQL/PostgreSQL] Deeb user: timeout`

**Solution**:
- Verify security groups allow your IP address
- Check VPN connection if required
- Verify RDS instance is in "available" state
- Check if there's a firewall blocking outbound connections

### No matching Deeb environment

**Error**: `No matching Deeb environment found for 'production'`

**Solution**:
- Create the environment in Deeb first
- Or update the `Environment` tag in the YAML to match an existing Deeb environment

### Authentication failed

**Error**: `Failed to load Deeb environments: 401 Unauthorized`

**Solution**:
- Verify your API token is correct and not expired
- Check that the Deeb API URL is correct

### Permission denied

**Error**: `permission denied to grant role`

**Solution**:
- Ensure the master user has sufficient privileges
- For PostgreSQL, the master user must own or have grant option on the roles

### PostgreSQL SSL requirement error

**Error**: `pg_hba.conf rejects connection... no encryption`

**Solution**:
This error occurs when registering PostgreSQL instances in Deeb with `require_ssl: False`, but PostgreSQL's `pg_hba.conf` requires SSL connections.

**Fix**: In the onboarding script, PostgreSQL instances should be registered with:
```python
"require_ssl": True,  # PostgreSQL RDS instances require SSL
```

Note: The script registers with `require_ssl: False` by default due to user modifications. You should change this back to `True` for PostgreSQL instances, or update your RDS `pg_hba.conf` to allow non-SSL connections (not recommended for production).

## Environment Tag Mapping

The scripts match RDS `Environment` tags to Deeb environments using case-insensitive comparison:

| RDS Tag | Deeb Environment |
|---------|------------------|
| `Production` | `production` |
| `production` | `production` |
| `PRODUCTION` | `production` |
| `Staging` | `staging` |

Make sure your Deeb environments are created before running the onboarding script.

## Security Considerations

- Deeb user passwords are randomly generated (32 characters)
- All database connections use SSL
- Master credentials are retrieved securely from AWS Secrets Manager
- Deeb stores encrypted connection details
- RDS IAM authentication is enabled for enhanced security

## Post-Onboarding

After successful onboarding:

1. Verify instances appear in Deeb UI under Database Instances
2. Test connectivity by creating a test change plan
3. Review the success/failure logs for any issues
4. Clean up: Keep the YAML config for reference or future re-runs

## Re-running the Scripts

The scripts are idempotent:
- Discovery script can be re-run to update the YAML with new instances
- Onboarding script will drop and recreate the Deeb user if it exists
- Deeb API will reject duplicate instance registrations (handle manually if needed)

## Support

For issues:
1. Check the failure log: `rds-onboarding-failed.log`
2. Review Deeb API documentation
3. Verify AWS permissions and network access
4. Check RDS instance status and configuration

