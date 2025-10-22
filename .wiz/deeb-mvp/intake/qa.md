# Q&A Session

**PRD**: deeb-mvp
**Created**: 2025-10-22T18:30:00Z
**Status**: Answers collected

## Questions and Answers

### 1. What are the primary language(s) for this project?

**Rationale**: Determines which design guidelines to generate and which language expert subagents to use during implementation

**Answer**: JavaScript for the frontend and Go for the backend

---

### 2. What is the benchmarking policy?

**Rationale**: Defines performance measurement requirements for critical code paths and optimization validation

**Answer**: Skip

---

### 3. What is the fuzzing policy?

**Rationale**: Determines fuzz testing scope for input validation and security-critical components

**Answer**: Core areas only

---

### 4. Who are the primary users of the Deeb MVP?

**Rationale**: Defines user personas to guide UI/UX design and feature prioritization

**Answer**: Backend engineers (Developers) and SREs

---

### 5. What are the top 3 pain points this MVP must solve?

**Rationale**: Ensures MVP focuses on highest-value problems and validates success criteria

**Answer**: Unsafe direct DB access, lack of change history, slow approval processes

---

### 6. What does success look like for the MVP launch?

**Rationale**: Establishes measurable goals for MVP validation and post-launch assessment

**Answer**: No manual apply of database changes with direct access to databases

---

### 7. Which database change types should the MVP support?

**Rationale**: Defines scope of SQL operations the system must handle, affecting parser and validator complexity

**Answer**: DDL and DML

---

### 8. Which approval workflow features are must-haves for MVP?

**Rationale**: Determines workflow engine complexity and approval process implementation scope

**Answer**: Simple single-approval and multi-stage apply

---

### 9. Which environments should the MVP support?

**Rationale**: Defines environment management requirements and promotion workflow complexity

**Answer**: Production and Staging

---

### 10. What policy validations are essential for MVP?

**Rationale**: Determines automated validation system scope and integration with SQL parsers/analyzers

**Answer**: Syntax, security checks, backwards compatibility, performance analysis, best practices

---

### 11. Should schema drift detection be included in MVP or deferred to post-MVP?

**Rationale**: Impacts system architecture for background jobs, monitoring, and alerting infrastructure

**Answer**: Post-MVP

---

### 12. What cloud platform(s) should be targeted?

**Rationale**: Determines infrastructure dependencies, authentication methods, and deployment strategies

**Answer**: AWS only with RDS focus

---

### 13. Are there specific frameworks/libraries preferred for the Web UI and API?

**Rationale**: Guides technology stack selection and ensures alignment with organizational standards

**Answer**: Frontend: React, Vite for dev, Shadcn for component libs (https://ui.shadcn.com). Backend: Go with net/http for API, spf13/cobra for server commands (https://github.com/spf13/cobra)

---

### 14. What database connection method should be prioritized for MVP?

**Rationale**: Affects database access layer design, credential management, and security implementation

**Answer**: RDS IAM auth and traditional username/password credentials

---

### 15. What are the expected concurrent users and database operations for MVP?

**Rationale**: Establishes performance requirements, infrastructure sizing, and scalability considerations

**Answer**: 100 users with 50 changes per day

---

### 16. Which authentication methods must be supported in MVP?

**Rationale**: Defines authentication system scope, user management complexity, and security posture

**Answer**: User email and password, OAuth 2.0 with Google only

---

### 17. What level of audit logging detail is required for MVP compliance needs?

**Rationale**: Determines audit system design, storage requirements, and compliance reporting capabilities

**Answer**: Comprehensive with user actions

---

### 18. What testing strategy should be prioritized for MVP?

**Rationale**: Establishes quality assurance scope, testing infrastructure needs, and CI/CD pipeline requirements

**Answer**: Most integration and e2e tests

---

### 19. What deployment model is expected for MVP?

**Rationale**: Impacts architecture decisions, multi-tenancy design, deployment automation, and operational support

**Answer**: Self-hosted in containerized deployment

---
