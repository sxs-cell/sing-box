//go:build with_utls

package tls

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net"
	"slices"
	"testing"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/logger"

	utls "github.com/metacubex/utls"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/hkdf"
)

func newRealityTestClient(t *testing.T, fingerprint string) (*RealityClientConfig, *ecdh.PrivateKey) {
	serverKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	config, err := NewRealityClient(context.Background(), logger.NOP(), "example.com", option.OutboundTLSOptions{
		Enabled:    true,
		ServerName: "example.com",
		UTLS:       &option.OutboundUTLSOptions{Enabled: true, Fingerprint: fingerprint},
		Reality: &option.OutboundRealityOptions{
			Enabled:   true,
			PublicKey: base64.RawURLEncoding.EncodeToString(serverKey.PublicKey().Bytes()),
			ShortID:   "0123456789abcdef",
		},
	})
	require.NoError(t, err)
	return config.(*RealityClientConfig), serverKey
}

func readRealityClientHello(t *testing.T, config *RealityClientConfig) ([]byte, *utls.PubClientHelloMsg) {
	client, server := net.Pipe()
	defer server.Close()
	go func() {
		defer client.Close()
		_, _ = config.ClientHandshake(context.Background(), client)
	}()
	var header [5]byte
	_, err := io.ReadFull(server, header[:])
	require.NoError(t, err)
	message := make([]byte, binary.BigEndian.Uint16(header[3:]))
	_, err = io.ReadFull(server, message)
	require.NoError(t, err)
	hello := utls.UnmarshalClientHello(message)
	require.NotNil(t, hello)
	return message, hello
}

func realityKeyShareIndex(hello *utls.PubClientHelloMsg, group utls.CurveID) int {
	return slices.IndexFunc(hello.KeyShares, func(share utls.KeyShare) bool {
		return share.Group == group
	})
}

// Xray-core >= 26.9.8 rejects a ClientHello without an X25519MLKEM768 key
// share ahead of the X25519 one, and authenticates from the X25519 share.
func TestRealityClientHelloX25519MLKEM768(t *testing.T) {
	t.Parallel()
	config, serverKey := newRealityTestClient(t, "chrome")
	message, hello := readRealityClientHello(t, config)

	mlkemIndex := slices.Index(hello.SupportedCurves, utls.X25519MLKEM768)
	require.GreaterOrEqual(t, mlkemIndex, 0)
	require.Less(t, mlkemIndex, slices.Index(hello.SupportedCurves, utls.X25519))
	mlkemIndex = realityKeyShareIndex(hello, utls.X25519MLKEM768)
	x25519Index := realityKeyShareIndex(hello, utls.X25519)
	require.GreaterOrEqual(t, mlkemIndex, 0)
	require.Less(t, mlkemIndex, x25519Index)

	peerKey, err := ecdh.X25519().NewPublicKey(hello.KeyShares[x25519Index].Data)
	require.NoError(t, err)
	authKey, err := serverKey.ECDH(peerKey)
	require.NoError(t, err)
	_, err = hkdf.New(sha256.New, authKey, hello.Random[:20], []byte("REALITY")).Read(authKey)
	require.NoError(t, err)
	block, err := aes.NewCipher(authKey)
	require.NoError(t, err)
	aead, err := cipher.NewGCM(block)
	require.NoError(t, err)
	additionalData := slices.Clone(message)
	clear(additionalData[39:71])
	plaintext, err := aead.Open(nil, hello.Random[20:], hello.SessionId, additionalData)
	require.NoError(t, err)
	require.Equal(t, []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}, plaintext[8:16])
}

// randomized picks the X25519MLKEM768 group and key share independently; the
// ClientHello keeps them exactly when the uTLS spec for the seed has both.
func TestRealityClientHelloRandomizedX25519MLKEM768(t *testing.T) {
	t.Parallel()
	config, _ := newRealityTestClient(t, "randomized")
	var kept int
	for range 64 {
		seed, err := utls.NewPRNGSeed()
		require.NoError(t, err)
		config.uClient.id.Seed = seed

		specConn, specPeer := net.Pipe()
		spec := utls.UClient(specConn, config.uClient.config.Clone(), config.uClient.id)
		require.NoError(t, spec.BuildHandshakeState())
		specConn.Close()
		specPeer.Close()
		var specCurve, specKeyShare bool
		for _, extension := range spec.Extensions {
			switch ext := extension.(type) {
			case *utls.SupportedCurvesExtension:
				specCurve = common.Contains(ext.Curves, utls.X25519MLKEM768)
			case *utls.KeyShareExtension:
				specKeyShare = common.Any(ext.KeyShares, func(share utls.KeyShare) bool {
					return share.Group == utls.X25519MLKEM768
				})
			}
		}
		keep := specCurve && specKeyShare

		_, hello := readRealityClientHello(t, config)
		require.Equal(t, keep, common.Contains(hello.SupportedCurves, utls.X25519MLKEM768))
		require.Equal(t, keep, realityKeyShareIndex(hello, utls.X25519MLKEM768) >= 0)
		if keep {
			kept++
		}
	}
	require.Positive(t, kept)
}
