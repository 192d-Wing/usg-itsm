// Package tlsconf centralizes TLS configuration so no service can accidentally
// weaken transport security. Per ADR-0007, only TLS 1.3 is permitted.
package tlsconf

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

// Base returns the hardened TLS 1.3-only configuration used everywhere.
func Base() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
	}
}

// Server returns a TLS 1.3 config presenting the given certificate.
func Server(cert tls.Certificate) *tls.Config {
	cfg := Base()
	cfg.Certificates = []tls.Certificate{cert}
	return cfg
}

// Client returns a TLS 1.3 client config for service-to-service calls. When
// caFile is set, only certificates chaining to that CA are trusted. When caFile
// is empty and allowInsecure is true (dev only), certificate verification is
// skipped. Otherwise the system trust store is used.
func Client(caFile string, allowInsecure bool) (*tls.Config, error) {
	cfg := Base()
	switch {
	case caFile != "":
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, errors.New("no certificates found in CA file")
		}
		cfg.RootCAs = pool
	case allowInsecure:
		cfg.InsecureSkipVerify = true //nolint:gosec // dev-only, gated by caller
	}
	return cfg, nil
}

// LoadServer loads a PEM cert/key pair into a TLS 1.3 server config.
func LoadServer(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load keypair: %w", err)
	}
	return Server(cert), nil
}

// SelfSigned generates an in-memory self-signed certificate for local
// development only. It must never be used outside a dev environment.
func SelfSigned(host string) (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host, Organization: []string{"usg-itsm-dev"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{host},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
		Leaf:        &tmpl,
	}, nil
}
