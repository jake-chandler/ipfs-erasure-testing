package ipfsclusterpeer

import (
	"fmt"
	"strings"
)

type ECFile struct {
	FileName        string
	CID             string
	ShardCIDs       []string
	ParityShardCIDs []string
	ClusterDAGCID   string
}

func newFileFromErasureDisabledOutput(output string) *ECFile {
	lines := strings.Split(output, "\n")
	if len(lines) < 1 {
		return nil
	}
	ECFile := &ECFile{}
	ECFile.CID = strings.Fields(lines[0])[1]
	ECFile.FileName = strings.Fields(lines[0])[2]
	return ECFile
}

func newECFileFromOutput(output string) *ECFile {
	lines := strings.Split(output, "\n")
	if len(lines) < 1 {
		return nil
	}

	ecFile := &ECFile{}

	// Parse the CID and other CIDs
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		cid := fields[1] // Middle field is the CID
		switch {
		case strings.Contains(line, "shard-") && !strings.Contains(line, "parity-shard-"):
			ecFile.ShardCIDs = append(ecFile.ShardCIDs, cid)
		case strings.Contains(line, "parity-shard-"):
			ecFile.ParityShardCIDs = append(ecFile.ParityShardCIDs, cid)
		case strings.Contains(line, "clusterDAG"):
			// This should be the line with the Cluster DAG CID
			ecFile.ClusterDAGCID = cid
		default:
			ecFile.FileName = fields[2]
			ecFile.CID = fields[1]
		}
	}

	return ecFile
}

func (ecf *ECFile) PrettyPrint() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Filename: %s\n", ecf.FileName))
	sb.WriteString(fmt.Sprintf("CID: %s\n", ecf.CID))
	sb.WriteString("Shard CIDs:\n")
	for _, shardCID := range ecf.ShardCIDs {
		sb.WriteString(fmt.Sprintf("\t%s\n", shardCID))
	}
	sb.WriteString("Parity Shard CIDs:\n")
	for _, parityCID := range ecf.ParityShardCIDs {
		sb.WriteString(fmt.Sprintf("\t%s\n", parityCID))
	}
	sb.WriteString(fmt.Sprintf("Cluster DAG CID: %s\n", ecf.ClusterDAGCID))

	return sb.String()
}
