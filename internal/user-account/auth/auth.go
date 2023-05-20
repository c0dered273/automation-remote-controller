package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

var (
	ownerOID             = asn1.ObjectIdentifier([]int{2, 5, 4, 32})
	x500UniqueIdentifier = asn1.ObjectIdentifier([]int{2, 5, 4, 45})
)

// JwtCustomClaims параметры которые хранятся jwt
type JwtCustomClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type ClientCert struct {
	Cert []byte
}

type CertKeyPair struct {
	Cert *x509.Certificate
	PKey any
}

// GenerateToken генерирует токен подписанный секретом
func GenerateToken(username string, secret string) (string, error) {
	claim := JwtCustomClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 720)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GetJWTConfig возвращает настройки для middleware которое проверяет jwt у входящих запросов
func GetJWTConfig(secret string) echojwt.Config {
	return echojwt.Config{
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(JwtCustomClaims)
		},
		SigningKey: []byte(secret),
	}
}

// GenerateCert генерирует пару сертификат/ключ для идентификации клиентского приложения
// возвращает pem блоки с сертификатом X.509 v3 и rsa приватным ключом в кодировке PKCS #8
// также в сертификат добавлены два объекта:
// owner - содержит имя пользователя telegram, которое использует хозяин клиентского приложения
// x500UniqueIdentifier - уникальный идентификатор клиентского приложения
func GenerateCert(
	caKeyPair CertKeyPair,
	ownerName string,
	clientID string,
	allowedDNS []string,
) (ClientCert, error) {
	clientCertPKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return ClientCert{}, err
	}
	clientCertPKeyBytes, err := x509.MarshalPKCS8PrivateKey(clientCertPKey)
	if err != nil {
		return ClientCert{}, err
	}

	clientCertTemplate := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixMicro()),
		Subject: pkix.Name{
			Country:      []string{"RU"},
			Organization: []string{"C0DERED"},
			Locality:     []string{"Moscow"},
			ExtraNames: []pkix.AttributeTypeAndValue{
				{
					Type:  ownerOID,
					Value: ownerName,
				},
				{
					Type:  x500UniqueIdentifier,
					Value: clientID,
				},
			},
		},
		NotBefore:                   time.Now().Add(-10 * time.Second),
		NotAfter:                    time.Now().AddDate(10, 0, 0),
		KeyUsage:                    x509.KeyUsageDigitalSignature,
		UnhandledCriticalExtensions: nil,
		ExtKeyUsage:                 []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		IsCA:                        false,
		DNSNames:                    allowedDNS,
		IPAddresses:                 []net.IP{net.IPv4(127, 0, 0, 1), net.IPv4(0, 0, 0, 0)},
	}

	clientCert, err := x509.CreateCertificate(rand.Reader, &clientCertTemplate, caKeyPair.Cert, &clientCertPKey.PublicKey, caKeyPair.PKey)
	if err != nil {
		return ClientCert{}, err
	}

	certPEM := bytes.Buffer{}
	err = pem.Encode(&certPEM, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: clientCertPKeyBytes,
	})
	if err != nil {
		return ClientCert{}, err
	}

	err = pem.Encode(&certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: clientCert,
	})
	if err != nil {
		return ClientCert{}, err
	}

	return ClientCert{
		Cert: certPEM.Bytes(),
	}, nil
}

// LoadKeyPair загружает с диска файлы сертификата и ключа и парсит их
func LoadKeyPair(certFile string, pkeyFile string) (CertKeyPair, error) {
	certPEMBytes, err := os.ReadFile(certFile)
	if err != nil {
		return CertKeyPair{}, err
	}
	pkeyPEMBytes, err := os.ReadFile(pkeyFile)
	if err != nil {
		return CertKeyPair{}, err
	}

	cert, err := ParseCert(certPEMBytes)
	if err != nil {
		return CertKeyPair{}, err
	}
	pkey, err := ParsePKey(pkeyPEMBytes)
	if err != nil {
		return CertKeyPair{}, err
	}

	return CertKeyPair{
		Cert: cert,
		PKey: pkey,
	}, nil
}

// ParseCert парсит pem блок в структуру сертификата
func ParseCert(certPEMBlock []byte) (*x509.Certificate, error) {
	for {
		var certDERBlock *pem.Block
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)

		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			clientCert, err := x509.ParseCertificate(certDERBlock.Bytes)
			if err != nil {
				return nil, err
			}
			return clientCert, nil
		}
	}

	return nil, errors.New("certificate not found")
}

// ParsePKey парсит pem блок в структуру приватного ключа
func ParsePKey(pkeyPEMBlock []byte) (any, error) {
	for {
		var pkeyDERBlock *pem.Block
		pkeyDERBlock, pkeyPEMBlock = pem.Decode(pkeyPEMBlock)

		if pkeyDERBlock == nil {
			break
		}
		if pkeyDERBlock.Type == "PRIVATE KEY" || strings.HasSuffix(pkeyDERBlock.Type, " PRIVATE KEY") {
			pkey, err := x509.ParsePKCS8PrivateKey(pkeyDERBlock.Bytes)
			if err != nil {
				return nil, err
			}
			return pkey, nil
		}
	}

	return nil, errors.New("private key not found")
}
