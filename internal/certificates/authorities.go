package certificates

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io"
	"os"
	"slices"
	"strings"
)

func LoadAuthorities(filename string) (*x509.CertPool, string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 65536 {
		return nil, "", errors.New("invalid CA file")
	}
	data, err := io.ReadAll(io.LimitReader(file, 65537))
	if err != nil || len(data) > 65536 {
		return nil, "", errors.New("invalid CA file size")
	}
	roots, err := x509.SystemCertPool()
	if err != nil {
		return nil, "", err
	}
	fingerprints := []string{}
	for len(bytes.TrimSpace(data)) > 0 {
		if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("-----BEGIN CERTIFICATE-----")) {
			return nil, "", errors.New("invalid CA PEM block")
		}
		block, remaining := pem.Decode(data)
		if block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			return nil, "", errors.New("invalid CA PEM certificate")
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil || !certificate.IsCA || !certificate.BasicConstraintsValid || certificate.KeyUsage != 0 && certificate.KeyUsage&x509.KeyUsageCertSign == 0 {
			return nil, "", errors.New("invalid certificate authority")
		}
		roots.AddCert(certificate)
		digest := sha256.Sum256(block.Bytes)
		fingerprints = append(fingerprints, hex.EncodeToString(digest[:]))
		data = remaining
	}
	if len(fingerprints) == 0 {
		return nil, "", errors.New("empty CA bundle")
	}
	slices.Sort(fingerprints)
	fingerprints = slices.Compact(fingerprints)
	digest := sha256.Sum256([]byte(strings.Join(fingerprints, "\n")))
	return roots, hex.EncodeToString(digest[:]), nil
}
