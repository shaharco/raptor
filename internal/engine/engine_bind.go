package engine

import (
	"encoding/json"
	"fmt"
	"github.com/natun-ai/natun/internal/plugin"
	"github.com/natun-ai/natun/pkg/api"
	manifests "github.com/natun-ai/natun/pkg/api/v1alpha1"
	"github.com/natun-ai/natun/pkg/errors"
)

// BindFeature converts the k8s Feature CRD to the internal implementation, and adds it to the engine.
func (e *engine) BindFeature(in manifests.Feature) error {
	md, err := api.MetadataFromManifest(in)
	if err != nil {
		return fmt.Errorf("failed to parse metadata from CR: %w", err)
	}

	ft := Feature{
		Metadata: *md,
	}

	if ft.Builder == "" {
		builderType := &manifests.FeatureBuilderType{}
		err := json.Unmarshal(in.Spec.Builder.Raw, builderType)
		if err != nil || builderType.Type == "" {
			return fmt.Errorf("failed to unmarshal builder type: %w", err)
		}
		ft.Builder = builderType.Type
	}

	if p := plugin.FeatureAppliers.Get(ft.Builder); p != nil {
		err := p(ft.Metadata, in.Spec.Builder.JSON.Raw, &ft, e)
		if err != nil {
			return err
		}
	}

	return e.bindFeature(ft)
}

func (e *engine) UnbindFeature(FQN string) error {
	e.features.Delete(FQN)
	e.logger.Info("feature unbound", "feature", FQN)
	return nil
}

func (e *engine) bindFeature(f Feature) error {
	if e.HasFeature(f.FQN) {
		return fmt.Errorf("%w: %s", errors.ErrFeatureAlreadyExists, f.FQN)
	}
	e.features.Store(f.FQN, f)
	e.logger.Info("feature bound", "FQN", f.FQN)
	return nil
}

func (e *engine) HasFeature(FQN string) bool {
	_, ok := e.features.Load(FQN)
	return ok
}