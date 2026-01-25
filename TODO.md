# Transaction Tracker - TODO List

## High Priority

### 1. Investigate and Fix Timeout Issues
- [ ] Review application logs to identify where timeouts are occurring
- [ ] Check if timeouts happen during:
  - OpenAI API calls
  - Notion API calls
  - HTTP request/response cycles
- [ ] Implement appropriate timeout configurations
- [ ] Add retry logic for external API calls
- [ ] Consider adding request timeout middleware

### 2. Implement Duplicate Transaction Prevention
- [ ] Fetch existing transactions from Notion before adding new ones
- [ ] Define duplicate detection criteria:
  - Same merchant/description
  - Same amount
  - Same date (or within X hours)
- [ ] Implement deduplication logic in transaction handler
- [ ] Add transaction hash/fingerprint for efficient comparison
- [ ] Handle edge cases (multiple legitimate transactions with same details)
- [ ] Add logging for skipped duplicate transactions
- [ ] Update API response to indicate which transactions were skipped as duplicates

### 3. Research Notion Alternatives
- [ ] Evaluate free database/storage alternatives:
  - Airtable (free tier limits?)
  - Google Sheets API
  - Supabase (PostgreSQL with REST API)
  - Firebase Firestore
  - MongoDB Atlas (free tier)
  - Coda API
- [ ] Compare based on:
  - Free tier limits (storage, API calls)
  - API ease of integration
  - Data export capabilities
  - UI for viewing data
  - Long-term sustainability
- [ ] Document findings and recommendation
- [ ] Create implementation plan for top alternative

## Future Enhancements

### Performance Optimizations
- [ ] Add caching layer for frequently accessed data
- [ ] Implement connection pooling for API clients
- [ ] Add rate limiting to prevent API quota exhaustion

### Monitoring & Logging
- [ ] Add structured logging
- [ ] Implement request/response logging
- [ ] Add metrics for API latency
- [ ] Set up error alerting

### Features
- [ ] Add transaction update/delete endpoints
- [ ] Implement bulk transaction import
- [ ] Add transaction search/filter endpoint
- [ ] Create monthly spending reports
- [ ] Add budget tracking and alerts

## Documentation
- [ ] Create CHANGELOG.md
- [ ] Document API error responses
- [ ] Add architecture diagram
- [ ] Create troubleshooting guide

---

**Last Updated:** 2026-01-25
