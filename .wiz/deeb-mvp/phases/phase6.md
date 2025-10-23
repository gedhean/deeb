# Phase 6: Deployment & Release

**Duration**: ~3 days (25 milestones @ 1h each)
**Dependencies**: Phase 5 (Testing, Security & Quality Assurance)
**Status**: 🚧 TODO

## Goal

Package the application for production deployment with Docker containers, create deployment automation with Kubernetes manifests and Docker Compose configurations, establish CI/CD pipeline, and prepare for MVP release. This phase ensures the system can be self-hosted in containerized environments with zero-downtime deployments and proper operational procedures.

## Phase-level Acceptance Criteria

- Dockerfile for Go backend optimized and multi-stage build working
- Dockerfile for React frontend with nginx serving static assets
- Docker images published to container registry (Docker Hub or AWS ECR)
- Docker Compose configuration for local development complete
- Docker Compose configuration for production deployment complete
- Kubernetes manifests for backend deployment created
- Kubernetes manifests for frontend deployment created
- Kubernetes ConfigMaps and Secrets management configured
- Kubernetes Ingress for HTTPS traffic routing configured
- Health check probes configured for rolling updates
- Database migration strategy for Kubernetes deployments documented
- CI/CD pipeline implemented (GitHub Actions or similar)
- Automated testing in CI pipeline operational
- Automated Docker image building and publishing working
- Zero-downtime deployment strategy tested (rolling updates)
- Database backup and restore procedures documented
- Monitoring setup guide created (Prometheus integration)
- Logging aggregation guide created
- Environment variable configuration guide complete
- Production deployment checklist created
- Rollback procedure documented
- Incident response runbook created
- Admin CLI commands documented (user management, database setup)
- Release notes prepared for MVP launch
- Post-deployment verification checklist created

## Milestones

<!-- Milestones appended by /wiz:milestones -->
### P06M01: Create Dockerfile for Go backend

**Status:** 🚧 TODO
**ID:** P06M01

**Goal**
Build optimized multi-stage Dockerfile for Go backend.

**Acceptance Criteria**
- [ ] Multi-stage build (builder and runtime)
- [ ] Alpine or distroless base image for small size
- [ ] Binary copied to runtime stage
- [ ] Non-root user configured
- [ ] Health check instruction included
- [ ] Image builds successfully and runs

---

### P06M02: Create Dockerfile for React frontend

**Status:** 🚧 TODO
**ID:** P06M02

**Goal**
Build Dockerfile for React frontend with nginx.

**Acceptance Criteria**
- [ ] Multi-stage build (build and serve)
- [ ] Vite build in build stage
- [ ] nginx Alpine image for serving
- [ ] Static assets copied to nginx html directory
- [ ] nginx configuration for SPA routing
- [ ] Image builds and serves correctly

---

### P06M03: Create Docker Compose for local development

**Status:** 🚧 TODO
**ID:** P06M03

**Goal**
Configure Docker Compose for local development environment.

**Acceptance Criteria**
- [ ] Services: backend, frontend, postgres
- [ ] Volume mounts for live reload
- [ ] Environment variables configured
- [ ] Network configuration for service communication
- [ ] PostgreSQL initialization scripts
- [ ] docker-compose up starts all services

---

### P06M04: Create Docker Compose for production

**Status:** 🚧 TODO
**ID:** P06M04

**Goal**
Configure Docker Compose for production deployment.

**Acceptance Criteria**
- [ ] Services: backend, frontend, postgres
- [ ] Production environment variables
- [ ] Resource limits configured
- [ ] Restart policies set to always
- [ ] Secrets management for credentials
- [ ] docker-compose up runs production stack

---

### P06M05: Publish Docker images to registry

**Status:** 🚧 TODO
**ID:** P06M05

**Goal**
Push Docker images to Docker Hub or AWS ECR.

**Acceptance Criteria**
- [ ] Images tagged with version and latest
- [ ] Push script created
- [ ] Registry authentication configured
- [ ] Images successfully pushed
- [ ] Image pull tested
- [ ] Documentation for pulling images

---

### P06M06: Create Kubernetes deployment for backend

**Status:** 🚧 TODO
**ID:** P06M06

**Goal**
Write Kubernetes Deployment manifest for Go backend.

**Acceptance Criteria**
- [ ] Deployment YAML with replica count
- [ ] Resource requests and limits
- [ ] Liveness and readiness probes
- [ ] Environment variables from ConfigMap
- [ ] Secrets for database credentials
- [ ] Deployment applies successfully

---

### P06M07: Create Kubernetes deployment for frontend

**Status:** 🚧 TODO
**ID:** P06M07

**Goal**
Write Kubernetes Deployment manifest for React frontend.

**Acceptance Criteria**
- [ ] Deployment YAML with replica count
- [ ] Resource requests and limits
- [ ] Liveness and readiness probes
- [ ] ConfigMap for frontend config
- [ ] Deployment applies successfully

---

### P06M08: Create Kubernetes Service for backend

**Status:** 🚧 TODO
**ID:** P06M08

**Goal**
Write Kubernetes Service manifest for backend API.

**Acceptance Criteria**
- [ ] Service YAML with ClusterIP type
- [ ] Port mapping (8080 -> backend port)
- [ ] Selector matches backend deployment
- [ ] Service applies and routes traffic correctly

---

### P06M09: Create Kubernetes Service for frontend

**Status:** 🚧 TODO
**ID:** P06M09

**Goal**
Write Kubernetes Service manifest for frontend.

**Acceptance Criteria**
- [ ] Service YAML with ClusterIP type
- [ ] Port mapping (80 -> frontend port)
- [ ] Selector matches frontend deployment
- [ ] Service applies and routes traffic correctly

---

### P06M10: Create Kubernetes ConfigMap

**Status:** 🚧 TODO
**ID:** P06M10

**Goal**
Write ConfigMap for application configuration.

**Acceptance Criteria**
- [ ] ConfigMap YAML with all config values
- [ ] Database connection settings (non-sensitive)
- [ ] Feature flags
- [ ] Log levels
- [ ] ConfigMap applies successfully
- [ ] Referenced by deployments

---

### P06M11: Create Kubernetes Secrets

**Status:** 🚧 TODO
**ID:** P06M11

**Goal**
Write Secret manifest for sensitive configuration.

**Acceptance Criteria**
- [ ] Secret YAML with base64-encoded values
- [ ] Database passwords
- [ ] JWT secret
- [ ] OAuth client secrets
- [ ] Secret applies successfully
- [ ] Referenced by deployments

---

### P06M12: Create Kubernetes Ingress

**Status:** 🚧 TODO
**ID:** P06M12

**Goal**
Write Ingress manifest for HTTPS traffic routing.

**Acceptance Criteria**
- [ ] Ingress YAML with TLS configuration
- [ ] Routes /api to backend service
- [ ] Routes / to frontend service
- [ ] TLS certificate configuration (cert-manager or manual)
- [ ] Ingress applies successfully
- [ ] Traffic routes correctly

---

### P06M13: Configure database migrations for Kubernetes

**Status:** 🚧 TODO
**ID:** P06M13

**Goal**
Set up Kubernetes Job for running database migrations.

**Acceptance Criteria**
- [ ] Job YAML for running migrations
- [ ] Runs before backend deployment
- [ ] Uses same database configuration
- [ ] Init container or Job pattern
- [ ] Migrations run successfully
- [ ] Documented in deployment guide

---

### P06M14: Create CI/CD pipeline with GitHub Actions

**Status:** 🚧 TODO
**ID:** P06M14

**Goal**
Configure GitHub Actions workflow for automated testing and building.

**Acceptance Criteria**
- [ ] Workflow triggers on push and PR
- [ ] Run Go tests
- [ ] Run frontend tests
- [ ] Run linting checks
- [ ] Build Docker images
- [ ] Workflow badge in README

---

### P06M15: Add Docker image building to CI/CD

**Status:** 🚧 TODO
**ID:** P06M15

**Goal**
Automate Docker image building and pushing in CI/CD.

**Acceptance Criteria**
- [ ] Build backend Docker image in CI
- [ ] Build frontend Docker image in CI
- [ ] Push to registry on main branch
- [ ] Tag with commit SHA and version
- [ ] Images available in registry
- [ ] Deployment uses CI-built images

---

### P06M16: Configure health check probes

**Status:** 🚧 TODO
**ID:** P06M16

**Goal**
Set up liveness and readiness probes for zero-downtime deployments.

**Acceptance Criteria**
- [ ] Liveness probe checks /health endpoint
- [ ] Readiness probe checks database connectivity
- [ ] Probe intervals and timeouts configured
- [ ] Rolling updates use probes correctly
- [ ] Zero-downtime deployment verified
- [ ] Documentation for probe configuration

---

### P06M17: Document database backup procedures

**Status:** 🚧 TODO
**ID:** P06M17

**Goal**
Create documentation for backing up and restoring metadata database.

**Acceptance Criteria**
- [ ] Backup procedure using pg_dump
- [ ] Automated backup script
- [ ] Backup retention policy (30 days)
- [ ] Restore procedure documented
- [ ] Test restore from backup
- [ ] Documentation in docs/

---

### P06M18: Create monitoring setup guide

**Status:** 🚧 TODO
**ID:** P06M18

**Goal**
Document Prometheus integration for monitoring.

**Acceptance Criteria**
- [ ] Prometheus scrape configuration
- [ ] Metrics endpoint documentation
- [ ] Sample Grafana dashboards (JSON)
- [ ] Alert rules for critical metrics
- [ ] Integration tested
- [ ] Documentation in docs/

---

### P06M19: Create logging aggregation guide

**Status:** 🚧 TODO
**ID:** P06M19

**Goal**
Document log aggregation setup for centralized logging.

**Acceptance Criteria**
- [ ] ELK stack setup guide
- [ ] Fluentd/Filebeat configuration
- [ ] Log parsing patterns
- [ ] Sample queries for common issues
- [ ] Integration tested
- [ ] Documentation in docs/

---

### P06M20: Document environment variable configuration

**Status:** 🚧 TODO
**ID:** P06M20

**Goal**
Create comprehensive guide for all environment variables.

**Acceptance Criteria**
- [ ] List of all environment variables
- [ ] Description and default value for each
- [ ] Required vs optional marked
- [ ] Examples for different environments
- [ ] Security considerations noted
- [ ] Documentation in docs/

---

### P06M21: Create production deployment checklist

**Status:** 🚧 TODO
**ID:** P06M21

**Goal**
Write pre-deployment checklist for production deployments.

**Acceptance Criteria**
- [ ] Database backup verified
- [ ] Environment variables configured
- [ ] SSL certificates valid
- [ ] Resource limits set
- [ ] Monitoring and alerting configured
- [ ] Rollback plan documented
- [ ] Checklist in docs/

---

### P06M22: Document rollback procedure

**Status:** 🚧 TODO
**ID:** P06M22

**Goal**
Create procedure for rolling back deployments.

**Acceptance Criteria**
- [ ] Kubernetes rollback commands
- [ ] Database migration rollback steps
- [ ] Docker image version pinning
- [ ] Testing rolled-back version
- [ ] Communication plan
- [ ] Documentation in docs/

---

### P06M23: Create incident response runbook

**Status:** 🚧 TODO
**ID:** P06M23

**Goal**
Document incident response procedures for operators.

**Acceptance Criteria**
- [ ] Incident severity levels
- [ ] On-call escalation procedures
- [ ] Common incident scenarios and resolutions
- [ ] Log analysis procedures
- [ ] Communication templates
- [ ] Runbook in docs/

---

### P06M24: Write release notes for MVP

**Status:** 🚧 TODO
**ID:** P06M24

**Goal**
Prepare release notes for MVP launch.

**Acceptance Criteria**
- [ ] Feature list with descriptions
- [ ] Known limitations documented
- [ ] Breaking changes (if any)
- [ ] Upgrade instructions (future releases)
- [ ] Credits and acknowledgments
- [ ] Release notes in CHANGELOG.md

---

### P06M25: Verify Phase 6 Completion

**Status:** 🚧 TODO
**ID:** P06M25

**Goal**
Verify that all Phase 6 requirements are met, deployment is functional, and the project is ready for MVP launch.

**Acceptance Criteria**
- [ ] All previous milestones in Phase 6 marked as complete
- [ ] Docker images built and published
- [ ] Kubernetes manifests deployed successfully
- [ ] CI/CD pipeline functional
- [ ] Zero-downtime deployment verified
- [ ] Monitoring and logging operational
- [ ] All documentation complete
- [ ] Production deployment checklist verified
- [ ] Rollback procedure tested
- [ ] Release notes prepared
- [ ] MVP ready for launch

---
