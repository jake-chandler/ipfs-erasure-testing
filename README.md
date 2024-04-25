# ipfs-erasure-testing
Testground plans for comparing ipfs-cluster &amp; ipfs-cluster-erasure

## Prerequisites

Before you begin, ensure that you have met the following requirements:

1. Install *Golang 1.16+*
2. Ensure Docker Daemon is running
3. Install Testground. See: https://github.com/testground/testground
4. Build the ipfs-cluster-erasure docker image. See: https://github.com/loomts/ipfs-cluster-erasure-example
5. Import the Test plans: `testground plan import --from ipfs-erasure-testing`

## Supported Builders / Runners
- As of 4/25/2024, the only supported builder/runner is `exec:go` and `local:exec`, respectively
- Implementation for a `docker:go` builder & `cluster:k8s` runner is currently in progress

## Test Cases
The test cases in this package are meant to analyze the following for comparing ipfs-cluster and ipfs-cluster-erasure:
- Efficiency
    - Disk Space Utilization
        - In theory, ipfs-cluster-erasure nodes should use less disk space than ifps-cluster
    - Additional pin overhead required (parity-shards, data-shards)
    - Performance
        - Do file pins take longer for ipfs-cluster-erasure?
            - How does file pinning operation scale with shard size?
- Fault Tolerance
    - Testing file pins / retrieval when nodes are in a failure state
        - Power Shut off
        - Network Bottlenecks
        - File I/O errors