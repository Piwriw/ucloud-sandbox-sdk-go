package template

import "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"

// Registry holds the credentials for pulling a base image from a private
// registry. Build one with BasicAuth, AWSAuth or GCPAuth; the zero value is not
// usable.
type Registry struct {
	inner api.FromImageRegistry
	err   error
}

// BasicAuth authenticates to a registry with a username and password.
func BasicAuth(username, password string) *Registry {
	r := &Registry{}
	r.err = r.inner.FromGeneralRegistry(api.GeneralRegistry{
		Type:     api.Registry,
		Username: username,
		Password: password,
	})
	return r
}

// AWSAuth authenticates to Amazon ECR.
func AWSAuth(accessKeyID, secretAccessKey, region string) *Registry {
	r := &Registry{}
	r.err = r.inner.FromAWSRegistry(api.AWSRegistry{
		Type:               api.Aws,
		AwsAccessKeyId:     accessKeyID,
		AwsSecretAccessKey: secretAccessKey,
		AwsRegion:          region,
	})
	return r
}

// GCPAuth authenticates to Google Artifact Registry with a service account key.
func GCPAuth(serviceAccountJSON string) *Registry {
	r := &Registry{}
	r.err = r.inner.FromGCPRegistry(api.GCPRegistry{
		Type:               api.Gcp,
		ServiceAccountJson: serviceAccountJSON,
	})
	return r
}

// apiValue returns the generated union, or the error encoding it produced.
func (r *Registry) apiValue() (*api.FromImageRegistry, error) {
	if r == nil {
		return nil, nil
	}
	if r.err != nil {
		return nil, r.err
	}
	inner := r.inner
	return &inner, nil
}
