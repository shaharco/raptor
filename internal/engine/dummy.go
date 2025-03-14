/*
Copyright (c) 2022 RaptorML authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package engine

import (
	"context"
	"github.com/raptor-ml/raptor/api"
	"time"
)

type Dummy struct {
	DataSource api.DataSource
	api.RuntimeManager
}

func (*Dummy) FeatureDescriptor(ctx context.Context, FQN string) (api.FeatureDescriptor, error) {
	return api.FeatureDescriptor{}, nil
}
func (*Dummy) Get(ctx context.Context, FQN string, keys api.Keys) (api.Value, api.FeatureDescriptor, error) {
	return api.Value{}, api.FeatureDescriptor{}, nil
}
func (*Dummy) Set(ctx context.Context, FQN string, keys api.Keys, val any, ts time.Time) error {
	return nil
}
func (*Dummy) Append(ctx context.Context, FQN string, keys api.Keys, val any, ts time.Time) error {
	return nil
}
func (*Dummy) Incr(ctx context.Context, FQN string, keys api.Keys, by any, ts time.Time) error {
	return nil
}
func (*Dummy) Update(ctx context.Context, FQN string, keys api.Keys, val any, ts time.Time) error {
	return nil
}

func (d *Dummy) GetDataSource(_ string) (api.DataSource, error) {
	return d.DataSource, nil
}
