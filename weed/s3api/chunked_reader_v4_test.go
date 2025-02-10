package s3api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"hash/crc32"

	"github.com/seaweedfs/seaweedfs/weed/s3api/s3err"
	"github.com/stretchr/testify/assert"
)

const (
	defaultTimestamp       = "20130524T000000Z"
	defaultBucketName      = "examplebucket"
	defaultAccessKeyId     = "AKIAIOSFODNN7EXAMPLE"
	defaultSecretAccessKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	defaultRegion          = "us-east-1"
)

func generatestreamingAws4HmacSha256Payload() string {
	// This test will implement the following scenario:
	// https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-streaming.html#example-signature-calculations-streaming

	chunk1 := "10000;chunk-signature=ad80c730a21e5b8d04586a2213dd63b9a0e99e0e2307b0ade35a65485a288648\r\n" +
		strings.Repeat("a", 65536) + "\r\n"
	chunk2 := "400;chunk-signature=0055627c9e194cb4542bae2aa5492e3c1575bbb81b612b7d234b86a503ef5497\r\n" +
		strings.Repeat("a", 1024) + "\r\n"
	chunk3 := "0;chunk-signature=b6c6ea8a5354eaf15b3cb7646744f4275b71ea724fed81ceb9323e279d449df9\r\n\r\n"

	payload := chunk1 + chunk2 + chunk3
	return payload
}

func NewRequeststreamingAws4HmacSha256Payload() (*http.Request, error) {
	// This test will implement the following scenario:
	// https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-streaming.html#example-signature-calculations-streaming

	payload := generatestreamingAws4HmacSha256Payload()
	req, err := http.NewRequest("PUT", "http://s3.amazonaws.com/examplebucket/chunkObject.txt", bytes.NewReader([]byte(payload)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Host", "s3.amazonaws.com")
	req.Header.Set("x-amz-date", defaultTimestamp)
	req.Header.Set("x-amz-storage-class", "REDUCED_REDUNDANCY")
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request,SignedHeaders=content-encoding;content-length;host;x-amz-content-sha256;x-amz-date;x-amz-decoded-content-length;x-amz-storage-class,Signature=4f232c4386841ef735655705268965c44a0e4690baa4adea153f7db9fa80a0a9")
	req.Header.Set("x-amz-content-sha256", "STREAMING-AWS4-HMAC-SHA256-PAYLOAD")
	req.Header.Set("Content-Encoding", "aws-chunked")
	req.Header.Set("x-amz-decoded-content-length", "66560")
	req.Header.Set("Content-Length", "66824")

	return req, nil
}

func TestNewSignV4ChunkedReaderstreamingAws4HmacSha256Payload(t *testing.T) {
	// This test will implement the following scenario:
	// https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-streaming.html#example-signature-calculations-streaming
	req, err := NewRequeststreamingAws4HmacSha256Payload()
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create an IdentityAccessManagement instance
	iam := IdentityAccessManagement{
		identities:        []*Identity{},
		accessKeyIdent:    map[string]*Identity{},
		accounts:          map[string]*Account{},
		emailAccount:      map[string]*Account{},
		hashes:            map[string]*sync.Pool{},
		hashCounters:      map[string]*int32{},
		identityAnonymous: nil,
		domain:            "",
		isAuthEnabled:     false,
	}

	// Add default access keys and secrets
	iam.identities = append(iam.identities, &Identity{
		Name: "default",
		Credentials: []*Credential{
			{
				AccessKey: defaultAccessKeyId,
				SecretKey: defaultSecretAccessKey,
			},
		},
		Actions: []Action{
			"Read",
			"Write",
			"List",
		},
	})

	iam.accessKeyIdent[defaultAccessKeyId] = iam.identities[0]

	// Call newSignV4ChunkedReader
	reader, errCode := iam.newSignV4ChunkedReader(req)
	assert.NotNil(t, reader)
	assert.Equal(t, s3err.ErrNone, errCode)

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read data: %v", err)
	}

	// The expected payload a long string of 'a's
	expectedPayload := strings.Repeat("a", 66560)
	assert.Equal(t, expectedPayload, string(data))

}



func generateStreamingUnsignedPayloadTrailerPayload() string {
	// This test will implement the following scenario:
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/checking-object-integrity.html

	chunk1 := "2000\r\n" + strings.Repeat("a", 8192) + "\r\n"
	chunk2 := "2000\r\n" + strings.Repeat("a", 8192) + "\r\n"
	chunk3 := "400\r\n" + strings.Repeat("a", 1024) + "\r\n"

	// ...

	data := strings.Repeat("a", 17408)
	checksum := crc32.ChecksumIEEE([]byte(data))
	checksumStr := fmt.Sprintf("%08x", checksum)
	trailer := "0\r\nx-amz-checksum-crc32:" + checksumStr + "\n\r\n\r\n\r\n"
	payload := chunk1 + chunk2 + chunk3 + trailer
	return payload
}

func NewRequestStreamingUnsignedPayloadTrailer() (*http.Request, error) {
	// This test will implement the following scenario:
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/checking-object-integrity.html

	payload := generateStreamingUnsignedPayloadTrailerPayload()
	req, err := http.NewRequest("PUT", "http://amzn-s3-demo-bucket/Key+", bytes.NewReader([]byte(payload)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Host", "amzn-s3-demo-bucket")
	req.Header.Set("x-amz-date", defaultTimestamp)
	req.Header.Set("Content-Encoding", "aws-chunked")
	req.Header.Set("x-amz-decoded-content-length", "17408")
	req.Header.Set("x-amz-content-sha256", "STREAMING-UNSIGNED-PAYLOAD-TRAILER")
	req.Header.Set("x-amz-trailer", "x-amz-checksum-crc32")

	return req, nil
}

func TestNewSignV4ChunkedReaderStreamingUnsignedPayloadTrailer(t *testing.T) {
	// This test will implement the following scenario:
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/checking-object-integrity.html
	req, err := NewRequestStreamingUnsignedPayloadTrailer()
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create an IdentityAccessManagement instance
	iam := IdentityAccessManagement{
		identities:        []*Identity{},
		accessKeyIdent:    map[string]*Identity{},
		accounts:          map[string]*Account{},
		emailAccount:      map[string]*Account{},
		hashes:            map[string]*sync.Pool{},
		hashCounters:      map[string]*int32{},
		identityAnonymous: nil,
		domain:            "",
		isAuthEnabled:     false,
	}

	// Add default access keys and secrets
	iam.identities = append(iam.identities, &Identity{
		Name: "default",
		Credentials: []*Credential{
			{
				AccessKey: defaultAccessKeyId,
				SecretKey: defaultSecretAccessKey,
			},
		},
		Actions: []Action{
			"Read",
			"Write",
			"List",
		},
	})

	iam.accessKeyIdent[defaultAccessKeyId] = iam.identities[0]

	// Call newSignV4ChunkedReader
	reader, errCode := iam.newSignV4ChunkedReader(req)
	assert.NotNil(t, reader)
	assert.Equal(t, s3err.ErrNone, errCode)

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read data: %v", err)
	}

	// The expected payload a long string of 'a's
	expectedPayload := strings.Repeat("a", 17408)
	assert.Equal(t, expectedPayload, string(data))
}
