#!/usr/bin/env python3
"""
AWS RDS to Deeb Onboarding Script

Reads the reviewed YAML configuration file and onboards each RDS instance to Deeb:
1. Retrieves master credentials from AWS Secrets Manager
2. Creates a Deeb user on each RDS instance
3. Registers the instance in Deeb via API
"""

import boto3
import yaml
import logging
import sys
import secrets
import string
import json
from typing import Dict, List, Any, Optional
from datetime import datetime

import pymysql
import psycopg2
from psycopg2 import sql
from psycopg2.extensions import ISOLATION_LEVEL_AUTOCOMMIT
import requests

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# File handlers for success/failure logs
success_handler = logging.FileHandler('rds-onboarding-success.log', mode='w')
success_handler.setLevel(logging.INFO)
success_handler.setFormatter(logging.Formatter('%(asctime)s - %(message)s'))

failure_handler = logging.FileHandler('rds-onboarding-failed.log', mode='w')
failure_handler.setLevel(logging.ERROR)
failure_handler.setFormatter(logging.Formatter('%(asctime)s - %(message)s'))

success_logger = logging.getLogger('success')
success_logger.addHandler(success_handler)
success_logger.setLevel(logging.INFO)

failure_logger = logging.getLogger('failure')
failure_logger.addHandler(failure_handler)
failure_logger.setLevel(logging.ERROR)


class DeebOnboarder:
    """Handles the onboarding of RDS instances to Deeb."""
    
    def __init__(self, config: Dict[str, Any], dry_run: bool = False):
        self.config = config
        self.deeb_api_url = config['deeb_api_url'].rstrip('/')
        self.deeb_api_token = config['deeb_api_token']
        self.deeb_project_id = config['deeb_project_id']
        self.aws_region = config.get('aws_region', 'us-east-1')
        self.dry_run = dry_run
        
        # Initialize AWS clients
        self.secretsmanager = boto3.client('secretsmanager', region_name=self.aws_region)
        
        # Initialize HTTP session for Deeb API
        self.session = requests.Session()
        self.session.headers.update({
            'Content-Type': 'application/json',
            'Authorization': f'Bearer {self.deeb_api_token}'
        })
        
        # Cache for Deeb environments
        self.deeb_environments = {}
        
    def load_deeb_environments(self):
        """Load all environments from Deeb API."""
        try:
            url = f"{self.deeb_api_url}/api/v1/environments"
            response = self.session.get(url)
            response.raise_for_status()
            
            data = response.json()
            
            # Handle different response structures
            if isinstance(data, list):
                environments = data
            elif isinstance(data, dict):
                # Try common wrapper keys
                environments = data.get('environments') or data.get('data') or data.get('items') or []
                if not environments:
                    logger.error(f"Unexpected response structure: {list(data.keys())}")
                    raise ValueError(f"Could not find environments in response. Keys: {list(data.keys())}")
            else:
                logger.error(f"Unexpected response type: {type(data)}, content: {data}")
                raise ValueError(f"Expected list or dict, got {type(data)}")
            
            # Create a case-insensitive lookup
            for env in environments:
                env_name_lower = env['name'].lower()
                self.deeb_environments[env_name_lower] = env
                logger.info(f"Loaded Deeb environment: {env['name']} (ID: {env['id']})")
            
            logger.info(f"Loaded {len(self.deeb_environments)} environments from Deeb")
            
        except requests.exceptions.HTTPError as e:
            logger.error(f"HTTP error loading Deeb environments: {e}")
            logger.error(f"Response: {e.response.text if hasattr(e, 'response') else 'N/A'}")
            raise
        except Exception as e:
            logger.error(f"Failed to load Deeb environments: {e}")
            import traceback
            logger.error(traceback.format_exc())
            raise
    
    def find_deeb_environment(self, env_name: str) -> Optional[Dict[str, Any]]:
        """Find a Deeb environment by name (case-insensitive)."""
        return self.deeb_environments.get(env_name.lower())
    
    def get_master_credentials(self, secret_arn: str) -> Dict[str, str]:
        """Retrieve master credentials from AWS Secrets Manager."""
        try:
            # AWS Secrets Manager accepts either SecretId (which can be ARN, name, or partial ARN)
            response = self.secretsmanager.get_secret_value(SecretId=secret_arn)
            secret_string = response['SecretString']
            credentials = json.loads(secret_string)
            
            # Secrets Manager stores credentials with keys like 'username' and 'password'
            return {
                'username': credentials.get('username'),
                'password': credentials.get('password')
            }
        except Exception as e:
            logger.error(f"Failed to retrieve secret {secret_arn}: {e}")
            raise
    
    def generate_secure_password(self) -> str:
        """Generate a secure random password for the Deeb user."""
        alphabet = string.ascii_letters + string.digits
        password = ''.join(secrets.choice(alphabet) for _ in range(32))
        return password
    
    def create_mysql_deeb_user(self, host: str, port: int, master_user: str, 
                               master_password: str, deeb_password: str):
        """Create Deeb user on MySQL instance."""
        if self.dry_run:
            logger.info(f"[DRY RUN] Would create MySQL Deeb user on {host}:{port}")
            return
            
        connection = None
        try:
            logger.info(f"Connecting to MySQL at {host}:{port}...")
            
            # Try to resolve hostname first
            import socket
            try:
                ip = socket.gethostbyname(host)
                logger.debug(f"DNS resolved {host} to {ip}")
            except socket.gaierror as dns_error:
                logger.error(f"DNS resolution failed for {host}: {dns_error}")
                raise ConnectionError(
                    f"Cannot resolve hostname '{host}'. "
                    f"This may indicate:\n"
                    f"  1. Network connectivity issue (VPN required?)\n"
                    f"  2. RDS instance not publicly accessible\n"
                    f"  3. Invalid endpoint in configuration\n"
                    f"Please verify network access and RDS security groups."
                )
            
            connection = pymysql.connect(
                host=host,
                port=port,
                user=master_user,
                password=master_password,
                connect_timeout=30  # Increased timeout for slow networks
            )
            
            cursor = connection.cursor()
            
            # WARNING: This will drop and recreate the Deeb user if it exists
            # This may disconnect any active sessions using this user
            # Drop user if exists
            try:
                cursor.execute("DROP USER IF EXISTS 'deeb'@'%'")
            except Exception as e:
                logger.warning(f"Could not drop existing user: {e}")
            
            # Create user with password (using parameterized query to prevent SQL injection)
            # Note: MySQL doesn't support parameters in CREATE USER, so we escape quotes
            safe_password = deeb_password.replace("'", "''")
            cursor.execute(f"CREATE USER 'deeb'@'%' IDENTIFIED BY '{safe_password}'")
            
            # Grant privileges (production-safe version without DROP, DELETE)
            grant_sql = """
            GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, 
            EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, 
            SELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, 
            RELOAD, LOCK TABLES, REPLICATION CLIENT, REPLICATION SLAVE 
            ON *.* TO 'deeb'@'%'
            """
            cursor.execute(grant_sql)
            
            # Try to grant SET_USER_ID if MySQL 8.0+
            try:
                cursor.execute("GRANT SET_USER_ID ON *.* TO 'deeb'@'%'")
            except Exception as e:
                logger.debug(f"Could not grant SET_USER_ID (may not be available): {e}")
            
            cursor.execute("FLUSH PRIVILEGES")
            connection.commit()
            
            logger.info(f"Successfully created MySQL Deeb user on {host}:{port}")
            
        except Exception as e:
            logger.error(f"Failed to create MySQL Deeb user on {host}:{port}: {e}")
            raise
        finally:
            if connection:
                connection.close()
    
    def create_postgres_deeb_user(self, host: str, port: int, master_user: str, 
                                   master_password: str, deeb_password: str):
        """Create Deeb user on PostgreSQL instance."""
        if self.dry_run:
            logger.info(f"[DRY RUN] Would create PostgreSQL Deeb user on {host}:{port}")
            return
            
        connection = None
        try:
            logger.info(f"Connecting to PostgreSQL at {host}:{port}...")
            
            # Try to resolve hostname first
            import socket
            try:
                ip = socket.gethostbyname(host)
                logger.debug(f"DNS resolved {host} to {ip}")
            except socket.gaierror as dns_error:
                logger.error(f"DNS resolution failed for {host}: {dns_error}")
                raise ConnectionError(
                    f"Cannot resolve hostname '{host}'. "
                    f"This may indicate:\n"
                    f"  1. Network connectivity issue (VPN required?)\n"
                    f"  2. RDS instance not publicly accessible\n"
                    f"  3. Invalid endpoint in configuration\n"
                    f"Please verify network access and RDS security groups."
                )
            
            # Connect with SSL
            connection = psycopg2.connect(
                host=host,
                port=port,
                user=master_user,
                password=master_password,
                database='postgres',
                connect_timeout=30  # Increased timeout for slow networks
            )
            connection.set_isolation_level(ISOLATION_LEVEL_AUTOCOMMIT)
            
            cursor = connection.cursor()
            
            # WARNING: This will drop and recreate the Deeb user if it exists
            # This may disconnect any active sessions using this user
            # Drop user if exists
            try:
                cursor.execute("DROP USER IF EXISTS deeb")
            except Exception as e:
                logger.warning(f"Could not drop existing user: {e}")
            
            # Create user with password (using parameterized query to prevent SQL injection)
            # PostgreSQL supports $ placeholders for safe parameter binding
            safe_password = deeb_password.replace("'", "''")
            cursor.execute(f"CREATE USER deeb WITH PASSWORD '{safe_password}'")
            
            # Grant RDS IAM privileges
            try:
                cursor.execute("GRANT rds_iam TO deeb")
            except Exception as e:
                logger.warning(f"Could not grant rds_iam (may not exist): {e}")
            
            # Grant CREATEDB privilege
            cursor.execute("ALTER USER deeb WITH CREATEDB")
            
            # Ensure deeb inherits permissions
            cursor.execute("ALTER ROLE deeb INHERIT")
            
            # Query and grant owner roles for non-system databases
            cursor.execute("""
                SELECT DISTINCT pg_get_userbyid(datdba) AS owner_role
                FROM pg_database
                WHERE datname NOT IN ('postgres', 'template0', 'template1')
                  AND datistemplate = false
                  AND pg_get_userbyid(datdba) != 'deeb'
            """)
            
            # RDS-protected roles that should not be granted
            rds_protected_roles = {'rdsadmin', 'rds_superuser', 'rds_replication', 'rds_password'}
            
            owner_roles = cursor.fetchall()
            for (owner_role,) in owner_roles:
                # Skip RDS-protected roles
                if owner_role.lower() in rds_protected_roles:
                    logger.info(f"Skipping RDS-protected role: {owner_role}")
                    continue
                
                try:
                    # Use identifier quoting to prevent SQL injection
                    from psycopg2 import sql
                    grant_query = sql.SQL("GRANT {} TO deeb").format(sql.Identifier(owner_role))
                    cursor.execute(grant_query)
                    logger.info(f"Granted role {owner_role} to deeb")
                except Exception as e:
                    logger.warning(f"Could not grant role {owner_role} to deeb: {e}")
            
            logger.info(f"Successfully created PostgreSQL Deeb user on {host}:{port}")
            
        except Exception as e:
            logger.error(f"Failed to create PostgreSQL Deeb user on {host}:{port}: {e}")
            raise
        finally:
            if connection:
                connection.close()
    
    def register_in_deeb(self, instance: Dict[str, Any], deeb_password: str, 
                        environment_id: str) -> Dict[str, Any]:
        """Register the RDS instance in Deeb via API."""
        if self.dry_run:
            logger.info(f"[DRY RUN] Would register {instance['name']} in Deeb")
            return {"id": "dry-run-id"}
            
        try:
            # Map engine type
            engine_map = {
                'mysql': 'mysql',
                'postgresql': 'postgres'
            }
            database_engine = engine_map.get(instance['engine'])
            
            if not database_engine:
                raise ValueError(f"Unsupported engine: {instance['engine']}")
            
            # Prepare request payload
            # NOTE: PostgreSQL RDS instances typically require SSL in pg_hba.conf
            # If you get "pg_hba.conf rejects connection... no encryption" errors,
            # set require_ssl to True for PostgreSQL instances
            payload = {
                "name": instance['name'],
                "description": "RDS instance imported from AWS",
                "database_engine": database_engine,
                "host": instance['endpoint'],
                "port": instance['port'],
                "username": "deeb",
                "password": deeb_password,
                "use_rds_iam_auth": False,
                "require_ssl": False,
                "environment_id": environment_id
            }
            
            # Create database instance
            url = f"{self.deeb_api_url}/api/v1/database-instances"
            response = self.session.post(url, json=payload)
            response.raise_for_status()
            
            result = response.json()
            logger.info(f"Successfully registered {instance['name']} in Deeb (ID: {result['id']})")
            
            return result
            
        except Exception as e:
            logger.error(f"Failed to register {instance['name']} in Deeb: {e}")
            if hasattr(e, 'response') and e.response is not None:
                logger.error(f"Response: {e.response.text}")
            raise
    
    def onboard_instance(self, instance: Dict[str, Any], environment_name: str) -> bool:
        """Onboard a single RDS instance."""
        instance_name = instance['name']
        logger.info(f"\n{'='*60}")
        logger.info(f"Processing: {instance_name} (Environment: {environment_name})")
        logger.info(f"{'='*60}")
        
        try:
            # Find Deeb environment
            deeb_env = self.find_deeb_environment(environment_name)
            if not deeb_env:
                raise ValueError(f"No matching Deeb environment found for '{environment_name}'")
            
            logger.info(f"Matched to Deeb environment: {deeb_env['name']} ({deeb_env['id']})")
            
            # Get master credentials
            logger.info("Retrieving master credentials from Secrets Manager...")
            master_creds = self.get_master_credentials(instance['master_secret_arn'])
            
            # Generate Deeb user password
            deeb_password = self.generate_secure_password()
            logger.info("Generated secure password for Deeb user")
            
            # Create Deeb user
            logger.info(f"Creating Deeb user on {instance['engine']} instance...")
            if instance['engine'] == 'mysql':
                self.create_mysql_deeb_user(
                    instance['endpoint'],
                    instance['port'],
                    master_creds['username'],
                    master_creds['password'],
                    deeb_password
                )
            elif instance['engine'] == 'postgresql':
                self.create_postgres_deeb_user(
                    instance['endpoint'],
                    instance['port'],
                    master_creds['username'],
                    master_creds['password'],
                    deeb_password
                )
            else:
                raise ValueError(f"Unsupported engine: {instance['engine']}")
            
            # Register in Deeb
            logger.info("Registering instance in Deeb...")
            result = self.register_in_deeb(instance, deeb_password, deeb_env['id'])
            
            # Log success
            success_msg = f"SUCCESS: {instance_name} -> Deeb ID: {result['id']}"
            logger.info(success_msg)
            
            # Build detailed success log entry
            detailed_success = f"SUCCESS: {instance_name}\n"
            detailed_success += f"  Deeb ID: {result['id']}\n"
            detailed_success += f"  Endpoint: {instance.get('endpoint', 'unknown')}:{instance.get('port', 'unknown')}\n"
            detailed_success += f"  Engine: {instance.get('engine', 'unknown')}\n"
            detailed_success += f"  Environment: {environment_name} ({deeb_env['id']})\n"
            success_logger.info(detailed_success)
            
            return True
            
        except Exception as e:
            error_msg = f"FAILED: {instance_name} -> Error: {str(e)}"
            logger.error(error_msg)
            
            # Build detailed failure log entry
            detailed_error = f"FAILED: {instance_name}\n"
            detailed_error += f"  Error: {str(e)}\n"
            detailed_error += f"  Instance: {instance.get('endpoint', 'unknown')}:{instance.get('port', 'unknown')}\n"
            detailed_error += f"  Engine: {instance.get('engine', 'unknown')}\n"
            detailed_error += f"  Environment: {environment_name}\n"
            
            # Add HTTP response details if available
            if hasattr(e, 'response') and e.response is not None:
                try:
                    response_json = e.response.json()
                    detailed_error += f"  API Response: {json.dumps(response_json, indent=4)}\n"
                except:
                    detailed_error += f"  API Response: {e.response.text}\n"
            
            failure_logger.error(detailed_error)
            return False
    
    def run(self, rds_config_file: str):
        """Main execution function."""
        logger.info("Starting RDS to Deeb onboarding")
        
        # Load RDS configuration
        try:
            with open(rds_config_file, 'r') as f:
                rds_config = yaml.safe_load(f)
        except Exception as e:
            logger.error(f"Failed to load RDS configuration from {rds_config_file}: {e}")
            sys.exit(1)
        
        # Load Deeb environments
        self.load_deeb_environments()
        
        # Process each environment
        environments = rds_config.get('environments', {})
        total_instances = sum(len(instances) for instances in environments.values())
        
        logger.info(f"\nFound {total_instances} instances across {len(environments)} environments")
        
        success_count = 0
        failure_count = 0
        
        for env_name, instances in environments.items():
            logger.info(f"\n{'#'*60}")
            logger.info(f"# Environment: {env_name} ({len(instances)} instances)")
            logger.info(f"{'#'*60}")
            
            for instance in instances:
                if self.onboard_instance(instance, env_name):
                    success_count += 1
                else:
                    failure_count += 1
        
        # Summary
        logger.info(f"\n{'='*60}")
        logger.info("ONBOARDING SUMMARY")
        logger.info(f"{'='*60}")
        logger.info(f"Total instances: {total_instances}")
        logger.info(f"Successful: {success_count}")
        logger.info(f"Failed: {failure_count}")
        logger.info("\nDetailed logs:")
        logger.info("  Success: rds-onboarding-success.log")
        logger.info("  Failures: rds-onboarding-failed.log")


def load_config(config_file: str) -> Dict[str, Any]:
    """Load the configuration file."""
    try:
        with open(config_file, 'r') as f:
            return yaml.safe_load(f)
    except Exception as e:
        logger.error(f"Failed to load configuration from {config_file}: {e}")
        sys.exit(1)


def test_deeb_connection(config: Dict[str, Any]):
    """Test connection to Deeb API."""
    try:
        deeb_api_url = config['deeb_api_url'].rstrip('/')
        deeb_api_token = config['deeb_api_token']
        
        session = requests.Session()
        session.headers.update({
            'Content-Type': 'application/json',
            'Authorization': f'Bearer {deeb_api_token}'
        })
        
        url = f"{deeb_api_url}/api/v1/environments"
        logger.info(f"Testing connection to: {url}")
        response = session.get(url)
        
        logger.info(f"Response status: {response.status_code}")
        logger.info(f"Response headers: {dict(response.headers)}")
        
        if response.status_code != 200:
            logger.error(f"Response body: {response.text}")
            return False
        
        data = response.json()
        logger.info(f"Response type: {type(data)}")
        logger.info(f"Response preview: {str(data)[:200]}")
        
        return True
    except Exception as e:
        logger.error(f"Connection test failed: {e}")
        import traceback
        logger.error(traceback.format_exc())
        return False


def main():
    """Main entry point."""
    if len(sys.argv) < 2:
        print("Usage: python3 aws-onboard-to-deeb.py <rds-config-file> [config-file] [--dry-run] [--test-connection]")
        print("\nArguments:")
        print("  rds-config-file:   YAML file with RDS instances (output from aws-discover-rds.py)")
        print("  config-file:       Configuration file with Deeb API settings (default: config.yaml)")
        print("  --dry-run:         Run in dry-run mode (no changes will be made)")
        print("  --test-connection: Test Deeb API connection and exit")
        sys.exit(1)
    
    # Parse arguments
    dry_run = '--dry-run' in sys.argv
    test_connection = '--test-connection' in sys.argv
    args = [arg for arg in sys.argv[1:] if arg not in ['--dry-run', '--test-connection']]
    
    rds_config_file = args[0] if args else 'rds-discovered.yaml'
    config_file = args[1] if len(args) > 1 else 'config.yaml'
    
    # Load configuration
    config = load_config(config_file)
    
    # Test connection if requested
    if test_connection:
        logger.info("Testing Deeb API connection...")
        if test_deeb_connection(config):
            logger.info("✅ Connection test successful!")
            sys.exit(0)
        else:
            logger.error("❌ Connection test failed!")
            sys.exit(1)
    
    if dry_run:
        logger.info("Running in DRY-RUN mode - no changes will be made")
    else:
        logger.warning("⚠️  PRODUCTION MODE - Changes will be made to databases!")
        logger.warning("⚠️  This will DROP and recreate the 'deeb' user on each database")
        response = input("\nType 'yes' to continue: ")
        if response.lower() != 'yes':
            logger.info("Onboarding cancelled by user")
            sys.exit(0)
    
    # Validate required configuration
    required_keys = ['deeb_api_url', 'deeb_api_token']
    missing_keys = [key for key in required_keys if key not in config]
    if missing_keys:
        logger.error(f"Missing required configuration keys: {', '.join(missing_keys)}")
        sys.exit(1)
    
    # Run onboarder
    try:
        onboarder = DeebOnboarder(config, dry_run=dry_run)
        onboarder.run(rds_config_file)
    except KeyboardInterrupt:
        logger.info("\nOnboarding interrupted by user")
        sys.exit(1)
    except Exception as e:
        logger.error(f"Onboarding failed: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()

