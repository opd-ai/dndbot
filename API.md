# DND Bot Generator API Documentation

## Overview
RESTful API for generating and managing D&D adventures. All endpoints support standard HTTP/HTTPS requests.

## Base URL
```
http://your-server/
```

## Core Endpoints

### Home Page
```http
GET /
```
Returns the main web interface.

**Response:**
- Content-Type: `text/html`
- Status: 200 OK

---

### Generate Adventure
```http
POST /generate
```
Initiates a new adventure generation process.

**Request:**
- Content-Type: `application/x-www-form-urlencoded`
- Body Parameters:
  - `prompt`: string (required) - The adventure generation prompt

**Response:**
- Status: 200 OK
- Headers:
  - `X-Session-Id`: Unique session identifier
- Set-Cookie: `session_id={uuid}; Path=/; HttpOnly; SameSite=Lax; MaxAge=86400`
- Content-Type: `text/html`

**Rate Limiting:**
- 3 requests per IP address per 4-hour window
- Status 429 if exceeded

**Error Responses:**
- 400 Bad Request: Invalid/missing prompt
- 429 Too Many Requests: Rate limit exceeded

---

### Get Messages History
```http
GET /api/messages/{sessionID}
```
Retrieves message history for a generation session.

**Parameters:**
- `sessionID`: UUID string (required) - Session identifier

**Response:**
- Status: 200 OK
- Content-Type: `text/html`
- Body: HTML-formatted message history

**Error Responses:**
- 404 Not Found: Invalid session ID
- 400 Bad Request: Malformed session ID

---

### Check Session Status
```http
GET /check-session
```
Validates session existence and status.

**Headers Required:**
- `X-Session-Id`: Session UUID
- OR Cookie: `session_id`

**Response:**
- Status: 200 OK
- Content-Type: `text/html`
- Body: Session status component

---

### Static Resources
```http
GET /static/*
```
Serves static assets (CSS, JS, images).

**Response:**
- Content-Type: Varies by resource type
- Cache-Control: public, max-age=3600

---

### Generated Outputs
```http
GET /outputs/*
```
Retrieves generated adventure files.

**Response:**
- Content-Type: Varies by file type
- Cache-Control: private, no-cache

## Authentication
- Session-based using secure cookies
- No additional authentication required
- Sessions expire after 24 hours

## CORS Configuration
```http
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Content-Type, X-Requested-With, HX-Request, HX-Current-URL
Access-Control-Allow-Credentials: true
Access-Control-Expose-Headers: X-Session-Id
```

## Error Handling
Standard HTTP status codes:
- 200: Success
- 400: Bad Request
- 404: Not Found
- 429: Too Many Requests
- 500: Server Error

## Security Features
- XSS Protection: All user inputs HTML-escaped
- CSRF Protection: SameSite cookie policy
- Rate Limiting: IP-based request throttling
- Secure Cookies: HttpOnly, SameSite=Lax
- Resource Protection: Restricted directory access

## Session Lifecycle
1. Created on first request
2. 24-hour validity
3. Automatic cleanup after 1 hour of inactivity
4. Cached for 24 hours after completion