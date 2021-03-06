package v1beta1

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/kubeflow/kfserving/pkg/constants"
	"github.com/kubeflow/kfserving/pkg/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AIXExplainerType string

const (
	AIXLimeImageExplainer AIXExplainerType = "LimeImages"
)

// AIXExplainerSpec defines the arguments for configuring an AIX Explanation Server
type AIXExplainerSpec struct {
	// The type of AIX explainer
	Type AIXExplainerType `json:"type"`
	// The location of a trained explanation model
	StorageURI string `json:"storageUri,omitempty"`
	// Defaults to latest AIX Version
	RuntimeVersion *string `json:"runtimeVersion,omitempty"`
	// Container enables overrides for the predictor.
	// Each framework will have different defaults that are populated in the underlying container spec.
	// +optional
	v1.Container `json:",inline"`
	// Inline custom parameter settings for explainer
	Config map[string]string `json:"config,omitempty"`
}

var _ ComponentImplementation = &AIXExplainerSpec{}

func (s *AIXExplainerSpec) GetStorageUri() *string {
	if s.StorageURI == "" {
		return nil
	}
	return &s.StorageURI
}

func (s *AIXExplainerSpec) GetResourceRequirements() *v1.ResourceRequirements {
	// return the ResourceRequirements value if set on the spec
	return &s.Resources
}

func (s *AIXExplainerSpec) GetContainer(metadata metav1.ObjectMeta, extensions *ComponentExtensionSpec, config *InferenceServicesConfig) *v1.Container {
	var args = []string{
		constants.ArgumentModelName, metadata.Name,
		constants.ArgumentPredictorHost, fmt.Sprintf("%s.%s", constants.DefaultPredictorServiceName(metadata.Name), metadata.Namespace),
		constants.ArgumentHttpPort, constants.InferenceServiceDefaultHttpPort,
	}
	if extensions.ContainerConcurrency != nil {
		args = append(args, constants.ArgumentWorkers, strconv.FormatInt(*extensions.ContainerConcurrency, 10))
	}
	if s.StorageURI != "" {
		args = append(args, "--storage_uri", constants.DefaultModelLocalMountPath)
	}

	args = append(args, "--explainer_type", string(s.Type))

	// Order explainer config map keys
	var keys []string
	for k, _ := range s.Config {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "--"+k)
		args = append(args, s.Config[k])
	}

	return &v1.Container{
		Image:     config.Explainers.AIXExplainer.ContainerImage + ":" + *s.RuntimeVersion,
		Name:      constants.InferenceServiceContainerName,
		Resources: s.Resources,
		Args:      args,
	}
}

func (s *AIXExplainerSpec) Default(config *InferenceServicesConfig) {
	s.Name = constants.InferenceServiceContainerName
	if s.RuntimeVersion == nil {
		s.RuntimeVersion = proto.String(config.Explainers.AIXExplainer.DefaultImageVersion)
	}
	setResourceRequirementDefaults(&s.Resources)
}

// Validate the spec
func (s *AIXExplainerSpec) Validate() error {
	return utils.FirstNonNilError([]error{
		validateStorageURI(s.GetStorageUri()),
	})
}
