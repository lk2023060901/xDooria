#!/bin/bash
# Generate TLS certificates for Kafka SASL/TLS testing
# Uses OpenSSL (macOS built-in, zero dependencies, CI/CD friendly)
# Requires: openssl, keytool (Java)

set -e

# Add Homebrew OpenJDK to PATH if available
if [ -d "/opt/homebrew/opt/openjdk/bin" ]; then
    export PATH="/opt/homebrew/opt/openjdk/bin:$PATH"
fi

# Check for keytool
if ! command -v keytool &> /dev/null; then
    echo "Error: keytool not found. Please install Java:"
    echo "  brew install openjdk"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERTS_DIR="${SCRIPT_DIR}/certs"

# Certificate validity (days)
VALIDITY=365

# Clean up existing certs
rm -rf "${CERTS_DIR}"
mkdir -p "${CERTS_DIR}"

echo "=== Generating CA certificate ==="
openssl req -new -x509 -days ${VALIDITY} -nodes \
    -keyout "${CERTS_DIR}/ca-key.pem" \
    -out "${CERTS_DIR}/ca-cert.pem" \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=Test/OU=Test/CN=TestCA"

echo "=== Generating Kafka server certificate ==="
# Generate server key
openssl genrsa -out "${CERTS_DIR}/server-key.pem" 2048

# Generate server CSR
openssl req -new \
    -key "${CERTS_DIR}/server-key.pem" \
    -out "${CERTS_DIR}/server.csr" \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=Test/OU=Test/CN=kafka"

# Create server extensions file for SAN
cat > "${CERTS_DIR}/server-ext.cnf" << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = kafka
DNS.2 = localhost
IP.1 = 127.0.0.1
EOF

# Sign server certificate with CA
openssl x509 -req -days ${VALIDITY} \
    -in "${CERTS_DIR}/server.csr" \
    -CA "${CERTS_DIR}/ca-cert.pem" \
    -CAkey "${CERTS_DIR}/ca-key.pem" \
    -CAcreateserial \
    -out "${CERTS_DIR}/server-cert.pem" \
    -extfile "${CERTS_DIR}/server-ext.cnf" \
    -extensions v3_req

echo "=== Generating client certificate ==="
# Generate client key
openssl genrsa -out "${CERTS_DIR}/client-key.pem" 2048

# Generate client CSR
openssl req -new \
    -key "${CERTS_DIR}/client-key.pem" \
    -out "${CERTS_DIR}/client.csr" \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=Test/OU=Test/CN=client"

# Sign client certificate with CA
openssl x509 -req -days ${VALIDITY} \
    -in "${CERTS_DIR}/client.csr" \
    -CA "${CERTS_DIR}/ca-cert.pem" \
    -CAkey "${CERTS_DIR}/ca-key.pem" \
    -CAcreateserial \
    -out "${CERTS_DIR}/client-cert.pem"

echo "=== Creating Kafka keystore and truststore ==="
# Convert server cert to PKCS12 format (for Kafka)
openssl pkcs12 -export \
    -in "${CERTS_DIR}/server-cert.pem" \
    -inkey "${CERTS_DIR}/server-key.pem" \
    -out "${CERTS_DIR}/server.p12" \
    -name kafka \
    -CAfile "${CERTS_DIR}/ca-cert.pem" \
    -caname root \
    -password pass:changeit

# Create Java keystore from PKCS12
keytool -importkeystore \
    -deststorepass changeit \
    -destkeypass changeit \
    -destkeystore "${CERTS_DIR}/kafka.keystore.jks" \
    -srckeystore "${CERTS_DIR}/server.p12" \
    -srcstoretype PKCS12 \
    -srcstorepass changeit \
    -alias kafka \
    -noprompt

# Create Java truststore with CA cert
keytool -importcert \
    -keystore "${CERTS_DIR}/kafka.truststore.jks" \
    -alias CARoot \
    -file "${CERTS_DIR}/ca-cert.pem" \
    -storepass changeit \
    -noprompt

# Clean up temporary files
rm -f "${CERTS_DIR}"/*.csr "${CERTS_DIR}"/*.cnf "${CERTS_DIR}"/*.srl "${CERTS_DIR}"/*.p12

echo "=== Certificates generated successfully ==="
echo ""
echo "Files created in ${CERTS_DIR}:"
ls -la "${CERTS_DIR}"
echo ""
echo "CA Certificate:     ${CERTS_DIR}/ca-cert.pem"
echo "Server Certificate: ${CERTS_DIR}/server-cert.pem"
echo "Server Key:         ${CERTS_DIR}/server-key.pem"
echo "Client Certificate: ${CERTS_DIR}/client-cert.pem"
echo "Client Key:         ${CERTS_DIR}/client-key.pem"
echo "Kafka Keystore:     ${CERTS_DIR}/kafka.keystore.jks"
echo "Kafka Truststore:   ${CERTS_DIR}/kafka.truststore.jks"
