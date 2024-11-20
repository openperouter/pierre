package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type DynamicASNMode string

const (
	InternalASNMode DynamicASNMode = "internal"
	ExternalASNMode DynamicASNMode = "external"
)

// Neighbor represents a BGP Neighbor we want FRR to connect to.
type Neighbor struct {
	// ASN is the AS number to use for the local end of the session.
	// ASN and DynamicASN are mutually exclusive and one of them must be specified.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=4294967295
	// +optional
	ASN uint32 `json:"asn,omitempty"`

	// DynamicASN detects the AS number to use for the local end of the session
	// without explicitly setting it via the ASN field. Limited to:
	// internal - if the neighbor's ASN is different than the router's the connection is denied.
	// external - if the neighbor's ASN is the same as the router's the connection is denied.
	// ASN and DynamicASN are mutually exclusive and one of them must be specified.
	// +kubebuilder:validation:Enum=internal;external
	// +optional
	DynamicASN DynamicASNMode `json:"dynamicASN,omitempty"`

	// SourceAddress is the IPv4 or IPv6 source address to use for the BGP
	// session to this neighbour, may be specified as either an IP address
	// directly or as an interface name
	// +optional
	SourceAddress string `json:"sourceaddress,omitempty"`

	// Address is the IP address to establish the session with.
	Address string `json:"address"`

	// Port is the port to dial when establishing the session.
	// Defaults to 179.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=16384
	Port *uint16 `json:"port,omitempty"`

	// Password to be used for establishing the BGP session.
	// Password and PasswordSecret are mutually exclusive.
	// +optional
	Password string `json:"password,omitempty"`

	// PasswordSecret is name of the authentication secret for the neighbor.
	// the secret must be of type "kubernetes.io/basic-auth", and created in the
	// same namespace as the frr-k8s daemon. The password is stored in the
	// secret as the key "password".
	// Password and PasswordSecret are mutually exclusive.
	// +optional
	PasswordSecret string `json:"passwordSecret,omitempty"`

	// HoldTime is the requested BGP hold time, per RFC4271.
	// Defaults to 180s.
	// +optional
	HoldTime *metav1.Duration `json:"holdTime,omitempty"`

	// KeepaliveTime is the requested BGP keepalive time, per RFC4271.
	// Defaults to 60s.
	// +optional
	KeepaliveTime *metav1.Duration `json:"keepaliveTime,omitempty"`

	// Requested BGP connect time, controls how long BGP waits between connection attempts to a neighbor.
	// +kubebuilder:validation:XValidation:message="connect time should be between 1 seconds to 65535",rule="duration(self).getSeconds() >= 1 && duration(self).getSeconds() <= 65535"
	// +kubebuilder:validation:XValidation:message="connect time should contain a whole number of seconds",rule="duration(self).getMilliseconds() % 1000 == 0"
	// +optional
	ConnectTime *metav1.Duration `json:"connectTime,omitempty"`

	// EBGPMultiHop indicates if the BGPPeer is multi-hops away.
	// +optional
	EBGPMultiHop bool `json:"ebgpMultiHop,omitempty"`

	// BFDProfile is the name of the BFD Profile to be used for the BFD session associated
	// to the BGP session. If not set, the BFD session won't be set up.
	// +optional
	BFDProfile string `json:"bfdProfile,omitempty"`

	// EnableGracefulRestart allows BGP peer to continue to forward data packets along
	// known routes while the routing protocol information is being restored. If
	// the session is already established, the configuration will have effect
	// after reconnecting to the peer
	// +optional
	EnableGracefulRestart bool `json:"enableGracefulRestart,omitempty"`

	// ToAdvertise represents the list of prefixes to advertise to the given neighbor
	// and the associated properties.
	// +optional
	ToAdvertise Advertise `json:"toAdvertise,omitempty"`

	// ToReceive represents the list of prefixes to receive from the given neighbor.
	// +optional
	ToReceive Receive `json:"toReceive,omitempty"`

	// To set if we want to disable MP BGP that will separate IPv4 and IPv6 route exchanges into distinct BGP sessions.
	// +optional
	// +kubebuilder:default:=false
	DisableMP bool `json:"disableMP,omitempty"`
}
