package ipfsclusterpeer

type ClusterPeer interface {
	GetPeerNumber() int
	GetIpfsContainerName() string
	GetClusterContainerName() string
	GetPeerIpfsPort() int
	GetPeerClusterPort() int
	GetPeerGatewayPort() int
	GetPeerSwarmPort() int
	TearDown() error
	LaunchNode() string
	StopNode() error
	StartNode() error
	GetFile(string) error
	ClearIPFSCache() error
	PrintPinnedFiles(string) error
	PinFile(string) (*ECFile, error)
}
