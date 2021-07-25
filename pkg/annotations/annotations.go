package annotations

import (
	"encoding/json"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

const OverridesV1Key = "port-map.mzg.io/overrides-v1"

type Annotations struct {
	Overrides Overrides
}

type Overrides map[PortDescriptor]*Override

type PortDescriptor struct {
	Port     int32
	Protocol corev1.Protocol
}

type Override struct {
	Skip bool
	Port int32 // 0 means no override
}

func FromService(service *corev1.Service) (*Annotations, error) {
	overrides := make(Overrides)

	if overridesData, ok := service.GetAnnotations()[OverridesV1Key]; ok {
		if err := json.Unmarshal([]byte(overridesData), &overrides); err != nil {
			return nil, err
		}
	}

	return &Annotations{Overrides: overrides}, nil
}

func (o Overrides) UnmarshalJSON(data []byte) error {
	var m map[string]*Override

	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	for k, v := range m {
		split := strings.SplitN(k, "/", 2) // nolint: gomnd
		if len(split) != 2 {               // nolint: gomnd
			return errors.Errorf("unable to split the key")
		}

		portInt, err := strconv.Atoi(split[1]) // nolint: gosec
		if err != nil {
			return err
		}

		pd := PortDescriptor{
			Protocol: corev1.Protocol(split[0]),
			Port:     int32(portInt),
		}

		o[pd] = v
	}
	return nil
}
