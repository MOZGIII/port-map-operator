package annotations

import (
	corev1 "k8s.io/api/core/v1"
)

type Annotations struct {
	Overrides Overrides
}

type Overrides map[PortDescriptor]*Override

type PortDescriptor struct {
	Port     uint16
	Protocol string
}

type Override struct {
	Skip bool
	Port uint16 // 0 means no override
}

func FromService(service *corev1.Service) (*Annotations, error) {
	overrides := make(Overrides)
	return &Annotations{Overrides: overrides}, nil
}
