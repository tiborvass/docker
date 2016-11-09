package convert

import (
	"github.com/Sirupsen/logrus"
	swarmtypes "github.com/tiborvass/docker/api/types/swarm"
	swarmapi "github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/protobuf/ptypes"
)

// SecretFromGRPC converts a grpc Secret to a Secret.
func SecretFromGRPC(s *swarmapi.Secret) swarmtypes.Secret {
	logrus.Debugf("%+v", s)
	secret := swarmtypes.Secret{
		ID:         s.ID,
		Digest:     s.Digest,
		SecretSize: s.SecretSize,
	}

	// Meta
	secret.Version.Index = s.Meta.Version.Index
	secret.CreatedAt, _ = ptypes.Timestamp(s.Meta.CreatedAt)
	secret.UpdatedAt, _ = ptypes.Timestamp(s.Meta.UpdatedAt)

	secret.Spec = &swarmtypes.SecretSpec{
		Annotations: swarmtypes.Annotations{
			Name:   s.Spec.Annotations.Name,
			Labels: s.Spec.Annotations.Labels,
		},
		Data: s.Spec.Data,
	}

	return secret
}

// SecretSpecToGRPC converts Secret to a grpc Secret.
func SecretSpecToGRPC(s swarmtypes.SecretSpec) (swarmapi.SecretSpec, error) {
	spec := swarmapi.SecretSpec{
		Annotations: swarmapi.Annotations{
			Name:   s.Name,
			Labels: s.Labels,
		},
		Data: s.Data,
	}

	return spec, nil
}
