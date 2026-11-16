/**
* @license
* Copyright 2026 Dynatrace LLC
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package vault

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/dynatrace-oss/terraform-provider-dynatrace/dynatrace/api"
	vault "github.com/dynatrace-oss/terraform-provider-dynatrace/dynatrace/api/platform/credential-vault/v1/settings"
	"github.com/dynatrace-oss/terraform-provider-dynatrace/dynatrace/rest"
	"github.com/dynatrace-oss/terraform-provider-dynatrace/dynatrace/settings"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/credentialvault"
)

func Service(clientSet rest.ClientSet) (settings.CRUDService[*vault.Credentials], error) {
	platformClient, err := clientSet.PlatformClient()
	if err != nil {
		return nil, err
	}

	return &service{client: credentialvault.NewClient(platformClient), envURL: clientSet.Credentials().Platform.EnvironmentURL}, nil
}

type service struct {
	client *credentialvault.Client
	envURL string
}

func (me *service) SchemaID() string {
	return "platform:credential-vault"
}

func (me *service) Create(ctx context.Context, v *vault.Credentials) (*api.Stub, error) {
	var data []byte
	var err error

	if data, err = json.Marshal(v); err != nil {
		return nil, err
	}

	if _, err = me.client.Create(ctx, data); err != nil {
		apiErr := coreapi.APIError{}
		if errors.As(err, &apiErr) {
			return nil, rest.Envelope(apiErr.Body, me.envURL, "POST")
		}
		return nil, err
	}

	return &api.Stub{Name: v.Name, ID: v.Name}, nil
}

func (me *service) Update(ctx context.Context, id string, v *vault.Credentials) error {
	var data []byte
	var err error

	if data, err = json.Marshal(v); err != nil {
		return err
	}

	_, err = me.client.Update(ctx, id, data)
	if err != nil {
		apiErr := coreapi.APIError{}
		if errors.As(err, &apiErr) {
			return rest.Envelope(apiErr.Body, me.envURL, "PUT")
		}
		return err
	}

	return err
}

func (me *service) Get(ctx context.Context, id string, v *vault.Credentials) error {
	err := me.get(ctx, id, v)
	if err != nil {
		return err
	}

	cfg := ctx.Value(settings.ContextKeyStateConfig)
	if stateCredentials, ok := cfg.(*vault.Credentials); ok {
		v.AllowContextlessRequests = stateCredentials.AllowContextlessRequests
		v.CertificateFormat = stateCredentials.CertificateFormat
		// and so on
		// ...
	}

	return nil
}

func (me *service) get(ctx context.Context, id string, v *vault.Credentials) (err error) {
	var result coreapi.Response
	if result, err = me.client.Get(ctx, id); err != nil {
		apiErr := coreapi.APIError{}
		if errors.As(err, &apiErr) {
			return rest.Envelope(apiErr.Body, me.envURL, "GET")
		}
		return err
	}

	return json.Unmarshal(result.Data, &v)
}

func (me *service) List(ctx context.Context) (api.Stubs, error) {
	result, err := me.client.List(ctx)
	if err != nil {
		return nil, err
	}
	var stubs api.Stubs
	for _, r := range result.All() {
		var credential vault.Credentials
		if err := json.Unmarshal(r, &credential); err != nil {
			return nil, err
		}
		stubs = append(stubs, &api.Stub{ID: credential.Name, Name: credential.Name}) // Name is the ID
	}
	return stubs, nil
}

func (me *service) Delete(ctx context.Context, id string) error {
	_, err := me.client.Delete(ctx, id)
	return err
}
