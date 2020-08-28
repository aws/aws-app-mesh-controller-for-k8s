#!/bin/bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

# $1=<CA name>
generate_ca() {
  openssl genrsa -out $DIR/$1_key.pem 2048
  openssl req -new -key $DIR/$1_key.pem -out $DIR/$1_cert.csr -config $DIR/$1_cert.cfg -batch -sha256
  openssl x509 -req -days 3650 -in $DIR/$1_cert.csr -signkey $DIR/$1_key.pem -out $DIR/$1_cert.pem \
    -extensions v3_ca -extfile $DIR/$1_cert.cfg
}

# $1=<certificate name>
generate_rsa_key() {
  openssl genrsa -out $DIR/$1_key.pem 2048
}

# $1=<certificate_name>
generate_cert_config() {
  sed -e "s/\${services_domain}/${SERVICES_DOMAIN}/" $DIR/$1_cert.cfg > $DIR/$1_cert.cfg.generated
}

# $1=<certificate name> $2=<CA name>
generate_x509_cert() {
  generate_cert_config $1
  openssl req -new -key $DIR/$1_key.pem -out $DIR/$1_cert.csr -config $DIR/$1_cert.cfg.generated -batch -sha256
  openssl x509 -req -days 3650 -in $DIR/$1_cert.csr -sha256 -CA $DIR/$2_cert.pem -CAkey \
    $DIR/$2_key.pem -CAcreateserial -out $DIR/$1_cert.pem -extensions v3_req -extfile $DIR/$1_cert.cfg.generated
}

# $1=<certificate name> $2=<CA name>
generate_cert_chain() {
  cat $DIR/$1_cert.pem $DIR/$2_cert.pem > $DIR/$1_cert_chain.pem
}

# $1=<CA name> $2=<CA name>
generate_cert_bundle() {
  cat $DIR/$1_cert.pem $DIR/$2_cert.pem > $DIR/$1_$2_bundle.pem
}

# Generate cert for the CA.
echo "Generating CA certificates."
generate_ca ca_1

# Generate RSA cert for the backend-tls app
echo "Generating backend certificate."
generate_rsa_key backend-tls ca_1
generate_x509_cert backend-tls ca_1
generate_cert_chain backend-tls ca_1

rm $DIR/*.csr
