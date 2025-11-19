# Deeb Helm Chart

This Helm chart deploys Deeb - a secure database change management platform - to Kubernetes.

## Prerequisites

- Kubernetes 1.20+
- Helm 3.8+
- Istio 1.16+ (if using Istio ingress)
- External Secrets Operator 0.9+ (if using AWS Secrets Manager)
- PostgreSQL (can be deployed via this chart as a dependency or use external instance)

## Installation

### Quick Start

1. Add the Helm repository (if published):
```bash
helm repo add deeb https://charts.deeb.io
helm repo update
```

2. Create a namespace:
```bash
kubectl create namespace deeb
```

3. Update Helm dependencies (to fetch PostgreSQL chart if enabled):
```bash
helm dependency update deeb/deeb
```

4. Install the chart:
```bash
helm install deeb deeb/deeb \
  --namespace deeb \
  --set global.domain=deeb.example.com \
  --set backend.image.repository=your-registry.io/deeb-backend \
  --set backend.image.tag=v1.0.0 \
  --set frontend.image.repository=your-registry.io/deeb-frontend \
  --set frontend.image.tag=v1.0.0
```

### Using External Secrets (AWS Secrets Manager)

1. Ensure External Secrets Operator is installed in your cluster:
```bash
kubectl get crd externalsecrets.external-secrets.io
```

2. Create a SecretStore or ClusterSecretStore (example for AWS):
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: aws-secrets-manager
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        jwt:
          serviceAccountRef:
            name: deeb-backend
            namespace: deeb
```

3. Create secrets in AWS Secrets Manager:

**Backend Secrets** (path: `deeb/backend`):
```json
{
  "jwt-secret": "your-jwt-secret-min-32-chars",
  "encryption-key": "your-32-byte-encryption-key",
  "database-password": "your-postgres-password",
  "google-client-id": "your-google-oauth-client-id",
  "google-client-secret": "your-google-oauth-client-secret",
  "github-client-id": "your-github-oauth-client-id",
  "github-client-secret": "your-github-oauth-client-secret"
}
```

**PostgreSQL Secrets** (path: `deeb/postgres`):
```json
{
  "postgres-password": "your-postgres-admin-password",
  "password": "your-postgres-user-password"
}
```

4. Configure IAM role for ServiceAccount (IRSA):
```yaml
backend:
  serviceAccount:
    annotations:
      eks.amazonaws.com/role-arn: arn:aws:iam::ACCOUNT_ID:role/deeb-backend
```

5. Install with External Secrets enabled:
```bash
helm install deeb deeb/deeb \
  --namespace deeb \
  --set externalSecrets.enabled=true \
  --set externalSecrets.secretStore.name=aws-secrets-manager \
  --set externalSecrets.secretStore.kind=ClusterSecretStore \
  --set externalSecrets.secrets.backend=deeb/backend \
  --set externalSecrets.secrets.postgres=deeb/postgres
```

### Using Embedded PostgreSQL

By default, the chart is configured to use an external PostgreSQL instance. To deploy PostgreSQL as part of this chart:

1. Update Helm dependencies to fetch the PostgreSQL chart:
```bash
helm dependency update deeb/deeb
```

2. Install with PostgreSQL enabled:
```bash
helm install deeb deeb/deeb \
  --namespace deeb \
  --set postgresql.enabled=true \
  --set externalSecrets.enabled=true \
  --set externalSecrets.secrets.postgres=deeb/postgres
```

The PostgreSQL instance will be deployed with:
- Persistent volume (10Gi by default)
- Security hardening (non-root, no privilege escalation)
- Metrics enabled for monitoring
- Credentials managed via External Secrets

**Note**: For production deployments, it's recommended to use a managed database service (like Amazon RDS or Google Cloud SQL) instead of the embedded PostgreSQL.

### Using External PostgreSQL

To use an external PostgreSQL instance:

1. Configure the external database connection:
```bash
helm install deeb deeb/deeb \
  --namespace deeb \
  --set postgresql.enabled=false \
  --set postgresql.externalHost=my-postgres.abc123.us-east-1.rds.amazonaws.com \
  --set postgresql.externalPort=5432 \
  --set postgresql.auth.username=deeb \
  --set postgresql.auth.database=deeb_metadata
```

2. Ensure the database password is available in your External Secrets backend under the configured path.

### Using Istio Ingress

1. Ensure Istio is installed:
```bash
kubectl get namespace istio-system
```

2. Create TLS certificate secret (if using HTTPS):
```bash
kubectl create secret tls deeb-tls-cert \
  --cert=path/to/cert.pem \
  --key=path/to/key.pem \
  -n deeb
```

3. Install with Istio enabled:
```bash
helm install deeb deeb/deeb \
  --namespace deeb \
  --set istio.enabled=true \
  --set istio.virtualService.hosts[0]=deeb.example.com \
  --set istio.virtualService.tls.enabled=true
```

## Configuration

### Global Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.domain` | Domain name for the application | `deeb.example.com` |
| `global.environment` | Environment name (dev/staging/production) | `production` |

### Backend Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `backend.replicaCount` | Number of backend replicas | `2` |
| `backend.image.repository` | Backend image repository | `your-registry.io/deeb-backend` |
| `backend.image.tag` | Backend image tag | Chart appVersion |
| `backend.image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `backend.resources.requests.cpu` | CPU request | `200m` |
| `backend.resources.requests.memory` | Memory request | `256Mi` |
| `backend.resources.limits.cpu` | CPU limit | `1000m` |
| `backend.resources.limits.memory` | Memory limit | `1Gi` |
| `backend.autoscaling.enabled` | Enable HPA | `true` |
| `backend.autoscaling.minReplicas` | Minimum replicas | `2` |
| `backend.autoscaling.maxReplicas` | Maximum replicas | `10` |

### Frontend Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `frontend.replicaCount` | Number of frontend replicas | `2` |
| `frontend.image.repository` | Frontend image repository | `your-registry.io/deeb-frontend` |
| `frontend.image.tag` | Frontend image tag | Chart appVersion |
| `frontend.resources.requests.cpu` | CPU request | `100m` |
| `frontend.resources.requests.memory` | Memory request | `128Mi` |
| `frontend.autoscaling.enabled` | Enable HPA | `true` |
| `frontend.autoscaling.minReplicas` | Minimum replicas | `2` |
| `frontend.autoscaling.maxReplicas` | Maximum replicas | `5` |

### PostgreSQL Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `postgresql.enabled` | Deploy PostgreSQL as part of this chart | `false` |
| `postgresql.auth.username` | PostgreSQL username | `deeb` |
| `postgresql.auth.database` | Database name | `deeb_metadata` |
| `postgresql.auth.existingSecret` | Existing secret for PostgreSQL credentials | `deeb-postgres-secret` |
| `postgresql.primary.persistence.enabled` | Enable persistence | `true` |
| `postgresql.primary.persistence.size` | PVC size | `10Gi` |
| `postgresql.primary.resources.limits.cpu` | CPU limit | `1000m` |
| `postgresql.primary.resources.limits.memory` | Memory limit | `1Gi` |
| `postgresql.externalHost` | External PostgreSQL host (when enabled=false) | `""` |
| `postgresql.externalPort` | External PostgreSQL port (when enabled=false) | `5432` |

### External Secrets Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `externalSecrets.enabled` | Enable External Secrets | `true` |
| `externalSecrets.refreshInterval` | Secret refresh interval | `1h` |
| `externalSecrets.secretStore.name` | SecretStore name | `aws-secrets-manager` |
| `externalSecrets.secretStore.kind` | SecretStore kind | `ClusterSecretStore` |
| `externalSecrets.secrets.backend` | Backend secrets path | `deeb/backend` |
| `externalSecrets.secrets.postgres` | PostgreSQL secrets path | `deeb/postgres` |

### Istio Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `istio.enabled` | Enable Istio VirtualService | `true` |
| `istio.gateway.create` | Create Istio Gateway | `false` |
| `istio.gateway.name` | Gateway name | `istio-gateway` |
| `istio.gateway.namespace` | Gateway namespace | `istio-system` |
| `istio.virtualService.hosts` | VirtualService hosts | `["deeb.example.com"]` |
| `istio.virtualService.tls.enabled` | Enable TLS | `true` |
| `istio.virtualService.cors.enabled` | Enable CORS | `true` |

## Upgrading

### From 0.1.x to 0.2.x

No breaking changes.

## Uninstalling

```bash
helm uninstall deeb --namespace deeb
```

To delete the namespace:
```bash
kubectl delete namespace deeb
```

## Troubleshooting

### Backend pods not starting

Check logs:
```bash
kubectl logs -n deeb -l app.kubernetes.io/name=deeb-backend
```

Common issues:
- Missing secrets (JWT_SECRET, ENCRYPTION_KEY, database password)
- Database connection issues
- Invalid configuration

### External Secrets not syncing

Check ExternalSecret status:
```bash
kubectl get externalsecrets -n deeb
kubectl describe externalsecret deeb-backend-secret -n deeb
```

Verify IAM permissions for the service account.

### Istio routing not working

Check VirtualService:
```bash
kubectl get virtualservice -n deeb
kubectl describe virtualservice deeb -n deeb
```

Verify Gateway exists:
```bash
kubectl get gateway -n istio-system
```

## Security Considerations

1. **Secrets Management**: Always use External Secrets Operator or sealed secrets for production deployments
2. **Network Policies**: Enable network policies to restrict pod-to-pod communication
3. **Pod Security Standards**: Pods run as non-root with read-only root filesystem
4. **RBAC**: Minimal ServiceAccount permissions are configured
5. **TLS**: Always enable TLS for production deployments
6. **Image Scanning**: Scan container images for vulnerabilities before deployment

## Support

- Documentation: https://docs.deeb.io
- Issues: https://github.com/NSXBet/deeb/issues
- Slack: https://deeb.slack.com

## License

Copyright © 2024 Deeb Team
