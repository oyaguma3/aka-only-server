# AKA Server User Guide

## Overview
This application is an API server that generates Milenage AKA authentication vectors for specified IMSIs. It is designed to work as a backend for EAP-AKA authentication.

## Prerequisites
- Go 1.21 or later
- PostgreSQL 14 or later

## Installation

1. Clone the repository.
2. Build the application:
   ```bash
   go build -o aka-server.exe ./cmd/server
   ```

## Database Setup

Ensure PostgreSQL is running and execute the following SQL to set up the database and user:

```sql
CREATE DATABASE akaserverdb;
\c akaserverdb
CREATE TABLE public.subscribers (
    imsi VARCHAR(15) PRIMARY KEY,
    ki   VARCHAR(32) NOT NULL,
    opc  VARCHAR(32) NOT NULL,
    sqn  VARCHAR(12) NOT NULL,
    amf  VARCHAR(4)  NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_imsi_format CHECK (imsi ~ '^[0-9]{15}$'),
    CONSTRAINT chk_ki_hex      CHECK (ki  ~ '^[0-9a-fA-F]{32}$'),
    CONSTRAINT chk_opc_hex     CHECK (opc ~ '^[0-9a-fA-F]{32}$'),
    CONSTRAINT chk_sqn_hex     CHECK (sqn ~ '^[0-9a-fA-F]{12}$'),
    CONSTRAINT chk_amf_hex     CHECK (amf ~ '^[0-9a-fA-F]{4}$')
);
CREATE USER akaserver WITH PASSWORD 'akaserver';
GRANT CONNECT ON DATABASE akaserverdb TO akaserver;
GRANT USAGE ON SCHEMA public TO akaserver;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO akaserver;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO akaserver;
```

## Configuration

Create a `.env` file in the same directory as the executable:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=akaserver
DB_PASSWORD=akaserver
DB_NAME=akaserverdb
API_PORT=8080
AUTH_API_ALLOWED_IPS=127.0.0.1,::1
DB_API_ALLOWED_IPS=127.0.0.1,::1
LOG_FILE=akaserver.log
LOG_MAX_SIZE=10
LOG_MAX_BACKUPS=3
LOG_MAX_AGE=28
```

## Running the Application

```bash
./aka-server.exe
```

## Systemd Service (Linux Only)

To run the application as a background service on Linux with systemd:

### Installation
Run the following command with root privileges:
```bash
sudo ./aka-server -install
```
This will:
1. Create a service file at `/etc/systemd/system/aka-server.service`.
2. Reload systemd.
3. Enable the service to start on boot.

You can then start the service:
```bash
sudo systemctl start aka-server
```

### Uninstallation
To remove the service:
```bash
sudo ./aka-server -uninstall
```

### Custom Service Name
You can specify a custom service name using the `-service-name` flag:
```bash
sudo ./aka-server -install -service-name my-aka-service
```

## API Usage

### 1. Create Subscriber
**POST** `/api/v1/subscribers`

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

### 2. Get Auth Vector (Normal)
**POST** `/api/v1/auth/{imsi}`

```bash
curl -X POST http://localhost:8080/api/v1/auth/123456789012345 \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Response:**
```json
{
    "rand": "...",
    "autn": "...",
    "xres": "...",
    "ck": "...",
    "ik": "..."
}
```

### 3. Get Auth Vector (Resync)
**POST** `/api/v1/auth/{imsi}`

```bash
curl -X POST http://localhost:8080/api/v1/auth/123456789012345 \
  -H "Content-Type: application/json" \
  -d '{
    "rand": "00000000000000000000000000000000",
    "auts": "0000000000000000000000000000"
  }'
```

### 4. Get Subscriber
**GET** `/api/v1/subscribers/{imsi}`

```bash
curl -X GET http://localhost:8080/api/v1/subscribers/123456789012345
```

### 5. Update Subscriber
**PUT** `/api/v1/subscribers/{imsi}`

```bash
curl -X PUT http://localhost:8080/api/v1/subscribers/123456789012345 \
  -H "Content-Type: application/json" \
  -d '{
    "ki": "00112233445566778899aabbccddeeff",
    "opc": "000102030405060708090a0b0c0d0e0f",
    "sqn": "000000000020",
    "amf": "8000"
  }'
```

### 6. Delete Subscriber
**DELETE** `/api/v1/subscribers/{imsi}`

```bash
curl -X DELETE http://localhost:8080/api/v1/subscribers/123456789012345
```

## Logging
Logs are written to `akaserver.log` (rotated automatically) and stdout.
