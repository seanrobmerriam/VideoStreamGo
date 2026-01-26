# AGENTS.md - VideoStreamGo AI Agent Integration Guide

## Overview

This document describes how AI agents can interact with the VideoStreamGo platform to assist with development, operations, customer support, and administrative tasks.

## Agent Capabilities

### 1. Development Assistant Agent

**Purpose**: Assist developers with code generation, debugging, and architecture decisions.

**Capabilities**:
- Generate boilerplate code for new handlers, services, and repositories
- Create database migrations based on schema requirements
- Write unit and integration tests
- Suggest optimal API endpoint structures
- Review code for best practices and security vulnerabilities
- Generate TypeScript types from Go models

**Example Interactions**:
```
User: "Create a new handler for video analytics tracking"
Agent: [Generates handler code with proper error handling, DTOs, and tests]

User: "Add a new field 'transcript' to the videos table"
Agent: [Creates migration file with proper up/down migrations]
```

### 2. DevOps Agent

**Purpose**: Assist with deployment, monitoring, and infrastructure management.

**Capabilities**:
- Generate Kubernetes manifests and Helm charts
- Create Docker Compose configurations
- Set up monitoring and alerting rules
- Troubleshoot deployment issues
- Optimize resource allocation
- Generate environment configuration templates

**Example Interactions**:
```
User: "Create a production-ready Kubernetes deployment for the platform-api"
Agent: [Generates deployment.yaml with proper resource limits, health checks, and security contexts]

User: "The instance-api pods are crashing, help me debug"
Agent: [Analyzes logs, checks configurations, suggests fixes]
```

### 3. Database Management Agent

**Purpose**: Manage database schemas, migrations, and queries.

**Capabilities**:
- Generate SQL migrations from model changes
- Optimize database queries and indexes
- Create backup and restore scripts
- Monitor database performance
- Suggest schema improvements for multi-tenancy
- Generate seed data for testing

**Example Interactions**:
```
User: "Create an index to optimize video search queries by category"
Agent: [Generates migration with appropriate index]

User: "Generate seed data for 5 test instances with 10 videos each"
Agent: [Creates SQL script with realistic test data]
```

### 4. Customer Support Agent

**Purpose**: Assist customer support team with technical inquiries.

**Capabilities**:
- Answer questions about platform features
- Troubleshoot instance provisioning issues
- Explain API usage and integration
- Generate custom integration examples
- Provide billing and subscription information
- Create customer-specific documentation

**Example Interactions**:
```
User: "Customer asks: How do I enable custom domains for my instance?"
Agent: [Provides step-by-step guide with API calls and configuration]

User: "Customer's video upload is failing with 413 error"
Agent: [Diagnoses file size limit issue, provides solution]
```

### 5. Instance Management Agent

**Purpose**: Automate tenant instance operations and management.

**Capabilities**:
- Provision new tenant instances
- Configure instance settings and branding
- Monitor instance health and usage
- Generate instance reports
- Handle instance migrations and backups
- Manage instance lifecycle (suspend, activate, terminate)

**Example Interactions**:
```
User: "Provision a new instance for customer 'acme-corp' with premium plan"
Agent: [Executes provisioning workflow, configures database, storage bucket, and DNS]

User: "Generate usage report for all instances from last month"
Agent: [Queries metrics, generates formatted report]
```

## Agent Integration Points

### API Integration

Agents can interact with VideoStreamGo through the Platform and Instance APIs:

**Platform API**: `http://localhost:8080/v1`
- Customer management
- Instance provisioning
- Billing operations
- Platform administration

**Instance API**: `http://localhost:8081/v1`
- Video management
- User operations
- Content moderation
- Analytics

### Authentication

Agents should authenticate using JWT tokens:

```bash
# Platform API authentication
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password"}'

# Use returned token in subsequent requests
curl -X GET http://localhost:8080/v1/customers \
  -H "Authorization: Bearer {token}"
```

### Database Access

Agents can access databases directly for read operations and analytics:

**Master Database**:
- Connection: `postgres://videostreamgo:securepassword@localhost:5432/videostreamgo_master`
- Purpose: Platform-level queries, customer analytics

**Instance Databases**:
- Pattern: `postgres://videostreamgo:securepassword@localhost:5432/instance_{subdomain}`
- Purpose: Tenant-specific operations, content queries

### Common Agent Tasks

#### Task 1: Create New Instance

```go
// Agent workflow
1. Create customer record via POST /customers
2. Create instance record via POST /instances
3. Provision database: instance_{subdomain}
4. Create storage bucket: {subdomain}-videos
5. Run instance migrations
6. Configure default branding
7. Set instance status to 'active'
```

#### Task 2: Generate API Client

```typescript
// Agent generates typed API client
interface VideoStreamGoClient {
  customers: CustomerAPI;
  instances: InstanceAPI;
  billing: BillingAPI;
  videos: VideoAPI;
}

// With full type safety from OpenAPI spec
```

#### Task 3: Monitor Instance Health

```bash
# Agent checks
1. Database connectivity
2. Storage bucket accessibility
3. Redis cache status
4. Video processing queue
5. API response times
6. Error rates
```

#### Task 4: Generate Migration

```sql
-- Agent creates migration from model changes
-- Migration: 20250126_add_video_chapters.sql

-- UP
ALTER TABLE videos ADD COLUMN chapters JSONB DEFAULT '[]';
CREATE INDEX idx_videos_chapters ON videos USING GIN (chapters);

-- DOWN
DROP INDEX idx_videos_chapters;
ALTER TABLE videos DROP COLUMN chapters;
```

## Agent Context Requirements

To effectively assist with VideoStreamGo, agents should have access to:

### Documentation Context
- Architecture overview (Section 2)
- API reference (Section 6)
- Database schema (Section 5)
- Configuration variables (Section 8)

### Codebase Context
- Project structure
- Naming conventions
- Error handling patterns
- Testing patterns

### Environment Context
- Deployment type (development/staging/production)
- Available resources
- External service configurations

### Customer Context
- Subscription plan
- Instance configuration
- Usage metrics
- Support history

## Agent Safety Guidelines

### Security Considerations
1. **Never expose sensitive data** in responses (passwords, API keys, tokens)
2. **Validate all inputs** before executing operations
3. **Use read-only access** when possible
4. **Log all agent actions** for audit trails
5. **Require approval** for destructive operations (delete, terminate)

### Data Privacy
1. **Respect tenant isolation** - don't mix customer data
2. **Anonymize examples** - use fake data in demonstrations
3. **Follow GDPR/compliance** requirements when handling user data

### Rate Limiting
1. Respect API rate limits (100 requests/hour by default)
2. Implement exponential backoff for retries
3. Use batch operations when available

## Agent Development Guidelines

### Creating a New Agent

```python
# Example: Video Analytics Agent

class VideoAnalyticsAgent:
    def __init__(self, api_client):
        self.client = api_client
        
    async def generate_report(self, instance_id, date_range):
        """Generate comprehensive video analytics report"""
        # 1. Fetch video data
        videos = await self.client.get_videos(instance_id)
        
        # 2. Aggregate metrics
        metrics = self.calculate_metrics(videos, date_range)
        
        # 3. Generate insights
        insights = self.generate_insights(metrics)
        
        # 4. Format report
        return self.format_report(metrics, insights)
    
    def calculate_metrics(self, videos, date_range):
        return {
            'total_views': sum(v.view_count for v in videos),
            'avg_duration': mean(v.duration for v in videos),
            'top_categories': self.get_top_categories(videos),
            'engagement_rate': self.calculate_engagement(videos)
        }
```

### Testing Agent Behavior

```python
# Unit tests for agent functionality
def test_agent_generate_report():
    agent = VideoAnalyticsAgent(mock_api_client)
    report = agent.generate_report(
        instance_id='test-instance',
        date_range=('2025-01-01', '2025-01-31')
    )
    
    assert 'total_views' in report
    assert 'insights' in report
    assert report['total_views'] >= 0
```

## Use Cases

### 1. Automated Instance Provisioning
**Trigger**: New customer signup
**Agent Actions**:
- Validate customer information
- Create database and storage
- Configure default settings
- Send welcome email with credentials

### 2. Intelligent Video Processing
**Trigger**: Video upload
**Agent Actions**:
- Validate video format
- Generate thumbnails
- Transcode to multiple resolutions
- Extract metadata
- Auto-categorize based on content

### 3. Proactive Monitoring
**Trigger**: Scheduled health checks
**Agent Actions**:
- Monitor instance performance
- Detect anomalies
- Alert on issues
- Suggest optimizations

### 4. Customer Support Automation
**Trigger**: Support ticket created
**Agent Actions**:
- Categorize issue
- Suggest relevant documentation
- Provide initial troubleshooting steps
- Escalate if needed

### 5. Analytics and Reporting
**Trigger**: End of billing period
**Agent Actions**:
- Aggregate usage metrics
- Generate invoices
- Create performance reports
- Identify upsell opportunities

## Agent Communication Protocol

### Request Format
```json
{
  "agent_id": "video-analytics-001",
  "task": "generate_report",
  "parameters": {
    "instance_id": "acme-corp",
    "date_range": {
      "start": "2025-01-01",
      "end": "2025-01-31"
    }
  },
  "context": {
    "user_role": "admin",
    "priority": "normal"
  }
}
```

### Response Format
```json
{
  "status": "success",
  "result": {
    "report_url": "https://reports.videostreamgo.com/abc123",
    "summary": {
      "total_views": 15420,
      "top_video": "Introduction to VideoStreamGo"
    }
  },
  "metadata": {
    "execution_time_ms": 3421,
    "timestamp": "2025-01-26T10:30:00Z"
  }
}
```

## Future Agent Capabilities

### Planned Enhancements
- **AI Content Moderation**: Automatically flag inappropriate content
- **Smart Recommendations**: Suggest related videos to users
- **Auto-Scaling**: Dynamically adjust resources based on demand
- **Predictive Maintenance**: Anticipate and prevent failures
- **Natural Language Queries**: Allow SQL-free database exploration
- **Automated Testing**: Generate and run test suites

## Contributing Agent Modules

To contribute a new agent module:

1. **Define the agent's purpose** and capabilities
2. **Create specification** document (follow this template)
3. **Implement core functionality** with tests
4. **Document API interactions** and data flows
5. **Submit pull request** with examples
6. **Update this AGENTS.md** with new capabilities

## Support and Resources

- **API Documentation**: See Section 6 of main documentation
- **Architecture Guide**: See Section 2 for system overview
- **Database Schema**: See Section 5 for data models
- **Configuration**: See Section 8 for environment setup

---

**Last Updated**: January 2025
**Version**: 1.0.0
**Maintained by**: VideoStreamGo Development Team