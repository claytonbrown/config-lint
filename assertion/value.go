package assertion

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// StandardValueSource can fetch values from external sources
type StandardValueSource struct{}

// GetValue looks up external values when an Expression includes a ValueFrom attribute
func (v StandardValueSource) GetValue(expression Expression) (string, error) {
	if expression.ValueFrom.URL != "" {
		Debugf("Getting value_from %s\n", expression.ValueFrom.URL)
		parsedURL, err := url.Parse(expression.ValueFrom.URL)
		if err != nil {
			return "", err
		}
		switch strings.ToLower(parsedURL.Scheme) {
		case "s3":
			return v.GetValueFromS3(parsedURL.Host, parsedURL.Path)
		case "http":
			return v.GetValueFromHTTP(expression.ValueFrom.URL)
		case "https":
			return v.GetValueFromHTTP(expression.ValueFrom.URL)
		default:
			return "", fmt.Errorf("Unsupported protocol for value_from: %s", parsedURL.Scheme)
		}
	}
	return expression.Value, nil
}

// GetValueFromS3 looks up external values for an Expression when the S3 protocol is specified
func (v StandardValueSource) GetValueFromS3(bucket string, key string) (string, error) {
	region := &aws.Config{Region: aws.String("us-east-1")}
	awsSession := session.New()
	s3Client := s3.New(awsSession, region)
	response, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	return strings.TrimSpace(buf.String()), nil
}

// GetValueFromHTTP looks up external value for an Expression when the HTTP protocol is specified
func (v StandardValueSource) GetValueFromHTTP(url string) (string, error) {
	httpResponse, err := http.Get(url)
	if err != nil {
		return "", err
	}
	if httpResponse.StatusCode != 200 {
		return "", err
	}
	defer httpResponse.Body.Close()
	body, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}
