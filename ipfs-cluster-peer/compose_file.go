package ipfsclusterpeer

const ComposeTemplatePeerN = `
version: '3.4'
services:
  ipfs{{PEER_NUMBER}}:
    container_name: ipfs{{PEER_NUMBER}}
    image: ipfs/kubo:release
    ports:
      - "4001" # ipfs swarm - expose if needed/wanted
      - "5001" # ipfs api - expose if needed/wanted
      - "8080" # ipfs gateway - expose if needed/wanted
    volumes:
      - ./compose/ipfs{{PEER_NUMBER}}:/data/ipfs
    networks:
      - ipfsnet

  cluster{{PEER_NUMBER}}:
    container_name: cluster{{PEER_NUMBER}}
    image: $IMAGE_NAME$
    depends_on:
      - ipfs{{PEER_NUMBER}}
    environment:
      CLUSTER_PEERNAME: cluster{{PEER_NUMBER}}
      CLUSTER_SECRET: "34a320169537634ea2b304eac9970d0203d94b320f82f09d89e96c94e2c7950c"
      CLUSTER_IPFSHTTP_NODEMULTIADDRESS: /dns4/ipfs{{PEER_NUMBER}}/tcp/5001
      CLUSTER_CRDT_TRUSTEDPEERS: '*' # Trust all peers in Cluster
      CLUSTER_RESTAPI_HTTPLISTENMULTIADDRESS: /ip4/0.0.0.0/tcp/9094 # Expose API
      CLUSTER_MONITORPINGINTERVAL: 2s # Speed up peer discovery
    ports:
      - "9095" # Cluster IPFS Proxy endpoint
      - "9096" # Cluster swarm endpoint
    volumes:
      - ./compose/cluster{{PEER_NUMBER}}:/data/ipfs-cluster
    networks:
      - ipfsnet
    command:
      - "daemon --bootstrap {{BOOTSTRAP_PEER}}"

networks:
  ipfsnet:
    external: true
`

const ComposeTempaltePeer0 = `
version: '3.4'
services:
  ipfs1:
    container_name: ipfs1
    image: ipfs/kubo:release
    volumes:
      - ./compose/ipfs1:/data/ipfs
    ports:
      - "5001"
      - "8080"
      - "4001"
    networks:
    - ipfsnet

  cluster1:
    container_name: cluster1
    image: $IMAGE_NAME$
    depends_on:
      - ipfs1
    environment:
      CLUSTER_PEERNAME: cluster1
      CLUSTER_SECRET: "34a320169537634ea2b304eac9970d0203d94b320f82f09d89e96c94e2c7950c" 
      CLUSTER_IPFSHTTP_NODEMULTIADDRESS: /dns4/ipfs1/tcp/5001
      CLUSTER_CRDT_TRUSTEDPEERS: '*' # Trust all peers in Cluster
      CLUSTER_RESTAPI_HTTPLISTENMULTIADDRESS: /ip4/0.0.0.0/tcp/9094 # Expose API
      CLUSTER_MONITORPINGINTERVAL: 2s # Speed up peer discovery
    ports:
          # Open API port (allows ipfs-cluster-ctl usage on host)
          - "127.0.0.1:9094:9094"
          # The cluster swarm port would need  to be exposed if this container
          # was to connect to cluster peers on other hosts.
          # But this is just a testing cluster.
          - "9095" # Cluster IPFS Proxy endpoint
          - "9096:9096" # Cluster swarm endpoint
    volumes:
      - ./compose/cluster1:/data/ipfs-cluster
    networks:
      - ipfsnet

networks:
  ipfsnet:
    external: true
`
