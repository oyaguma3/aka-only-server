# AKA Server API Specification

## Overview
This document describes the REST API provided by the AKA Server.
The API allows for subscriber management and AKA authentication vector generation.

## Base URL
`http://<host>:<port>/api/v1`

## Authentication & Security
- **IP Allowlist**: Access is restricted based on the client IP address as configured in the `.env` file.
    - `AUTH_API_ALLOWED_IPS`: Controls access to Authentication endpoints.
    - `DB_API_ALLOWED_IPS`: Controls access to Subscriber Management endpoints.
    - **Multiple IPs**: You can specify multiple IP addresses separated by commas.

    **Configuration Example (.env):**
    ```env
    AUTH_API_ALLOWED_IPS=127.0.0.1,::1,192.168.1.100
    DB_API_ALLOWED_IPS=127.0.0.1,10.0.0.5
    ```

## Endpoints

### 1. Authentication

#### Generate Authentication Vector
Generates a Milenage AKA authentication vector for a specific subscriber. Supports both normal authentication and resynchronization.

- **URL**: `/auth/:imsi`
- **Method**: `POST`
- **URL Params**:
    - `imsi` (Required): The IMSI of the subscriber (15 digits).

##### Request Body (Normal Authentication)
Send an empty JSON object.
```json
{}
```

##### Request Body (Resynchronization)
Send `rand` and `auts` received from the USIM.
```json
{
    "rand": "00000000000000000000000000000000",
    "auts": "0000000000000000000000000000"
}
```
- `rand`: 32-character hex string (16 bytes).
- `auts`: 28-character hex string (14 bytes).

##### Success Response (200 OK)
Returns the generated authentication vector.
```json
{
    "rand": "00000000000000000000000000000000",
    "autn": "00000000000000000000000000000000",
    "xres": "0000000000000000",
    "ck":   "00000000000000000000000000000000",
    "ik":   "00000000000000000000000000000000"
}
```
- All fields are hex strings.

##### Error Responses
- `404 Not Found`: Subscriber not found.
- `500 Internal Server Error`: Database error or AKA calculation failure.

---

### 2. Subscriber Management

#### Create Subscriber
Registers a new subscriber in the database.

- **URL**: `/subscribers`
- **Method**: `POST`

##### Request Body
```json
{
    "imsi": "123456789012345",
    "ki":   "00112233445566778899aabbccddeeff",
    "opc":  "000102030405060708090a0b0c0d0e0f",
    "sqn":  "000000000000",
    "amf":  "8000"
}
```
- `imsi`: 15 digits.
- `ki`: 32 hex characters (16 bytes).
- `opc`: 32 hex characters (16 bytes).
- `sqn`: 12 hex characters (6 bytes).
- `amf`: 4 hex characters (2 bytes).

##### Success Response (201 Created)
Empty body.

##### Error Responses
- `400 Bad Request`: Invalid input format.
- `500 Internal Server Error`: Database error (e.g., duplicate IMSI).

#### Get Subscriber Count
Returns the total number of registered subscribers.

- **URL**: `/subscribers/count`
- **Method**: `GET`

##### Success Response (200 OK)
```json
{
    "count": 100
}
```

##### Error Responses
- `500 Internal Server Error`: Database error.

#### List Subscribers
Returns a list of all registered subscribers.

- **URL**: `/subscribers`
- **Method**: `GET`

##### Success Response (200 OK)
```json
[
    {
        "imsi": "123456789012345",
        "ki":   "00112233445566778899aabbccddeeff",
        "opc":  "000102030405060708090a0b0c0d0e0f",
        "sqn":  "000000000020",
        "amf":  "8000",
        "created_at": "2023-10-27T10:00:00Z"
    },
    ...
]
```

##### Error Responses
- `500 Internal Server Error`: Database error.

#### Get Subscriber
Retrieves subscriber details.

- **URL**: `/subscribers/:imsi`
- **Method**: `GET`
- **URL Params**:
    - `imsi` (Required): The IMSI of the subscriber.

##### Success Response (200 OK)
```json
{
    "imsi": "123456789012345",
    "ki":   "00112233445566778899aabbccddeeff",
    "opc":  "000102030405060708090a0b0c0d0e0f",
    "sqn":  "000000000020",
    "amf":  "8000",
    "created_at": "2023-10-27T10:00:00Z"
}
```

##### Error Responses
- `404 Not Found`: Subscriber not found.
- `500 Internal Server Error`: Database error.

#### Update Subscriber
Updates an existing subscriber's credentials.

- **URL**: `/subscribers/:imsi`
- **Method**: `PUT`
- **URL Params**:
    - `imsi` (Required): The IMSI of the subscriber.

##### Request Body
```json
{
    "ki":   "00112233445566778899aabbccddeeff",
    "opc":  "000102030405060708090a0b0c0d0e0f",
    "sqn":  "000000000020",
    "amf":  "8000"
}
```
Note: `imsi` in the body is ignored; the URL parameter is used.

##### Success Response (200 OK)
Empty body.

##### Error Responses
- `400 Bad Request`: Invalid input format.
- `500 Internal Server Error`: Database error.

#### Delete Subscriber
Removes a subscriber from the database.

- **URL**: `/subscribers/:imsi`
- **Method**: `DELETE`
- **URL Params**:
    - `imsi` (Required): The IMSI of the subscriber.

##### Success Response (204 No Content)
Empty body.

##### Error Responses
- `500 Internal Server Error`: Database error.

---

## Example Usage (curl)

### 1. Create Subscriber

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/subscribers \
  -H "Content-Type: application/json" \
  -d '{
    "imsi": "123456789012345",
    "ki": "00112233445566778899aabbccddeeff",
    "opc": "000102030405060708090a0b0c0d0e0f",
    "sqn": "000000000000",
    "amf": "8000"
  }'
```

**Response (201 Created):**
(No Content)

### 2. Get Subscriber Count

**Request:**
```bash
curl -X GET http://localhost:8080/api/v1/subscribers/count
```

**Response (200 OK):**
```json
{
    "count": 42
}
```

### 3. List Subscribers

**Request:**
```bash
curl -X GET http://localhost:8080/api/v1/subscribers
```

**Response (200 OK):**
```json
[
    {
        "imsi": "123456789012345",
        ...
    },
    ...
]
```

### 4. Get Auth Vector (Normal)

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/123456789012345 \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Response (200 OK):**
```json
{
    "rand": "d83d3b9a...",
    "autn": "a1b2c3d4...",
    "xres": "1a2b3c4d...",
    "ck": "e5f6g7h8...",
    "ik": "9i0j1k2l..."
}
```

### 3. Get Auth Vector (Resync)

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/123456789012345 \
  -H "Content-Type: application/json" \
  -d '{
    "rand": "00000000000000000000000000000000",
    "auts": "0000000000000000000000000000"
  }'
```

**Response (200 OK):**
```json
{
    "rand": "00000000000000000000000000000000",
    "autn": "f1e2d3c4...",
    "xres": "5a6b7c8d...",
    "ck": "9e8f7g6h...",
    "ik": "1i2j3k4l..."
}
```

### 4. Get Subscriber

**Request:**
```bash
curl -X GET http://localhost:8080/api/v1/subscribers/123456789012345
```

**Response (200 OK):**
```json
{
    "imsi": "123456789012345",
    "ki": "00112233445566778899aabbccddeeff",
    "opc": "000102030405060708090a0b0c0d0e0f",
    "sqn": "000000000020",
    "amf": "8000",
    "created_at": "2023-11-23T12:00:00Z"
}
```

### 5. Update Subscriber

**Request:**
```bash
curl -X PUT http://localhost:8080/api/v1/subscribers/123456789012345 \
  -H "Content-Type: application/json" \
  -d '{
    "ki": "00112233445566778899aabbccddeeff",
    "opc": "000102030405060708090a0b0c0d0e0f",
    "sqn": "000000000040",
    "amf": "8000"
  }'
```

**Response (200 OK):**
(No Content)

### 6. Delete Subscriber

**Request:**
```bash
curl -X DELETE http://localhost:8080/api/v1/subscribers/123456789012345
```

**Response (204 No Content):**
(No Content)
