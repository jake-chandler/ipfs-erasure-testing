package ipfsclusterpeer

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/testground/sdk-go/runtime"
)

// BaseTemplateFile is the path to the template file
const BaseTemplateFile = "/home/jake/testground/plans/ipfstesting/compose_file_generator/cluster-peer-compose-template.yml"
const Peer0TemplateFile = "/home/jake/testground/plans/ipfstesting/compose_file_generator/cluster-peer0-compose-template.yml"
const TestFileDirectory = "/home/jake/testground/plans/ipfstesting/test_files"

type IpfsClusterPeer struct {
	PeerNumber int
	Template   string // Template field to store the compose template content

	// this channel recieves the peer id once the node has successfully launched
	PeerIdChannel  *chan string
	runenv         *runtime.RunEnv
	bootstrapOnId  string
	containerID    string
	erasureEnabled bool
}

// New creates a new IpfsClusterPeerHelper instance - if bootstrap id is blank/nil, we are assumed to be the first peer
func New(peerNumber int, runenv *runtime.RunEnv, bootstrapId string) (*IpfsClusterPeer, error) {
	if peerNumber != 1 && bootstrapId == "" {
		return nil, fmt.Errorf("peer %d requires a bootstrapId", peerNumber)
	}
	// Read the template file
	var bootstrapOnId string = bootstrapId
	var template string
	if peerNumber == 1 {
		template = Peer0TemplateFile
	} else {
		template = BaseTemplateFile
	}
	templateContent, err := os.ReadFile(template)
	if err != nil {
		return nil, err
	}
	c := make(chan string)
	return &IpfsClusterPeer{
		PeerNumber:     peerNumber,
		Template:       string(templateContent),
		runenv:         runenv,
		PeerIdChannel:  &c,
		bootstrapOnId:  bootstrapOnId,
		erasureEnabled: runenv.BooleanParam("erasureEnabled"),
	}, nil
}

// GenerateComposeFile generates a Docker Compose file content based on the peer number
func (c *IpfsClusterPeer) GenerateComposeFile() error {
	composeContent := strings.ReplaceAll(c.Template, "{{PEER_NUMBER}}", strconv.Itoa(c.PeerNumber))
	composeContent = strings.ReplaceAll(composeContent, "{{PEER_IPFS_PORT}}", strconv.Itoa(c.GetPeerIpfsPort()))
	composeContent = strings.ReplaceAll(composeContent, "{{PEER_CLUSTER_PORT}}", strconv.Itoa(c.GetPeerClusterPort()))
	composeContent = strings.ReplaceAll(composeContent, "{{PEER_SWARM_PORT}}", strconv.Itoa(c.GetPeerSwarmPort()))
	composeContent = strings.ReplaceAll(composeContent, "{{PEER_GATEWAY_PORT}}", strconv.Itoa(c.GetPeerGatewayPort()))

	var imageName string = "ipfs/ipfs-cluster:latest"
	if c.erasureEnabled {
		imageName = "ipfs-cluster-erasure:latest"
	}
	composeContent = strings.ReplaceAll(composeContent, "$IMAGE_NAME$", imageName)
	if c.bootstrapOnId != "" {
		peer := fmt.Sprintf("/dns4/%s/tcp/9096/ipfs/%s", fmt.Sprintf("cluster%d", 1), c.bootstrapOnId)
		composeContent = strings.ReplaceAll(composeContent, "{{BOOTSTRAP_PEER}}", peer)
	}

	return c.writeToFile(composeContent)
}

// the directory name where this peer's docker-compose.yml file will live
func (c *IpfsClusterPeer) GetPeerDockerDirectory() string {
	return fmt.Sprintf("peer%d", c.PeerNumber)
}

func (c *IpfsClusterPeer) GetIpfsContainerName() string {
	return fmt.Sprintf("ipfs%d", c.PeerNumber)
}
func (c *IpfsClusterPeer) GetClusterContainerName() string {
	return fmt.Sprintf("cluster%d", c.PeerNumber)
}
func (c *IpfsClusterPeer) getAnyPort(base int) int {
	// Ensure peerNumber is within the range of 1 to 99
	subport := (c.PeerNumber + 99) % 100
	// Add a base port number to the peerNumber to get a unique port
	port := base + subport
	return port + 1
}
func (c *IpfsClusterPeer) GetPeerIpfsPort() int {
	return c.getAnyPort(5000)
}
func (c *IpfsClusterPeer) GetPeerClusterPort() int {
	return c.getAnyPort(9000)
}
func (c *IpfsClusterPeer) GetPeerGatewayPort() int {
	return c.getAnyPort(8000)
}
func (c *IpfsClusterPeer) GetPeerSwarmPort() int {
	return c.getAnyPort(4000)
}

// writeToFile writes the generated compose content to a file named docker-compose.yml in a directory named after the peer number
func (c *IpfsClusterPeer) writeToFile(content string) error {
	// Create a directory named after the peer number
	dirName := c.GetPeerDockerDirectory()
	if err := os.MkdirAll(dirName, 0755); err != nil {
		return err
	}
	if c.runenv.RunParams.BooleanParam("verbose") {
		c.runenv.RecordMessage(content)
	} // Write the compose content to docker-compose.yml in the created directory
	return os.WriteFile(fmt.Sprintf("%s/docker-compose.yml", dirName), []byte(content), 0644)
}

// extractPeerID extracts the peer ID from a line that contains "IPFS is ready. Peer ID" pattern
func extractPeerID(line string) string {
	// Split the line by spaces
	parts := strings.Split(line, " ")
	for i, part := range parts {
		// Check if the current part is "Peer" and the next part exists
		if part == "ID:" && i+1 < len(parts) {
			return strings.TrimSpace(parts[i+1]) // Return the next part (peer ID)
		}
	}
	return ""
}
func getContainerIDByName(containerName string) (string, error) {
	// Run `docker ps -aq --filter "name=<containerName>"` to get the container ID
	cmd := exec.Command("docker", "ps", "-aq", "--filter", "name="+containerName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Trim any leading or trailing whitespace from the output
	containerID := strings.TrimSpace(string(output))

	if containerID == "" {
		return "", fmt.Errorf("container '%s' not found", containerName)
	}

	return containerID, nil
}

func killContainer(containerName string) error {
	// Get the ID of the container by name
	containerID, err := getContainerIDByName(containerName)
	if err != nil {
		return err
	}

	// Kill the container using `docker kill <containerID>`
	cmd := exec.Command("docker", "kill", containerID)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Delete the container using `docker rm <containerID>`
	cmd = exec.Command("docker", "rm", containerID)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
func (c *IpfsClusterPeer) TearDown() error {
	c.runenv.RecordMessage("Peer %d: Tearing Down", c.PeerNumber)
	for _, containerName := range []string{c.GetClusterContainerName(), c.GetIpfsContainerName()} {
		if err := killContainer(containerName); err != nil {
			c.runenv.RecordMessage("%s", fmt.Errorf("error killing container %s", err.Error()))
			return err
		}
	}
	c.runenv.RecordMessage("Peer %d: Successfully Terminated Containers", c.PeerNumber)
	o, _ := executeCommand(exec.Command("pwd"))
	c.runenv.RecordMessage("\npwd output:\n\n")
	c.runenv.RecordMessage(o)
	return nil
}

func (c *IpfsClusterPeer) LaunchNode() error {
	err := c.GenerateComposeFile()
	if err != nil {
		c.runenv.RecordMessage("Failure generating compose file")
		c.runenv.RecordFailure(err)
		return err
	}
	err = os.Chdir(c.GetPeerDockerDirectory())
	if err != nil {
		return err
	}

	cmd := exec.Command("docker-compose", "up")
	// Create a pipe for capturing command output
	stdout, err := cmd.StderrPipe()
	if err != nil {
		*c.PeerIdChannel <- ""
		c.runenv.RecordFailure(err)
		return fmt.Errorf("error creating stdout pipe: %s", err)
	}

	// Read command output line by line and send it to the output channel
	reader := bufio.NewReader(stdout)
	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					*c.PeerIdChannel <- ""
				}
				break
			}
			// Check if the line contains the desired pattern
			if strings.Contains(line, "IPFS is ready.") {

				// Extract the peer ID from the line and send it through the peer ID channel
				peerID := extractPeerID(line)
				if peerID != "" {
					*c.PeerIdChannel <- peerID
					break
				}
			}
		}
	}()
	// Start the command
	if err := cmd.Start(); err != nil {
		c.runenv.RecordFailure(err)
		*c.PeerIdChannel <- ""
		return fmt.Errorf("error starting command: %s", err)
	}
	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		c.runenv.RecordFailure(err)
		*c.PeerIdChannel <- ""
		return err
	}
	return nil
}

func stopOrStartContainerByName(containerName string, start bool) (string, error) {
	// Get the ID of the container by name
	containerID, err := getContainerIDByName(containerName)
	if err != nil {
		return "", err
	}

	// Stop the container using `docker stop <containerID>`
	var command string
	if start {
		command = "start"
	} else {
		command = "stop"
	}
	cmd := exec.Command("docker", command, containerID)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return containerID, nil
}
func stopContainerByName(containerName string) (string, error) {
	cid, err := stopOrStartContainerByName(containerName, false)
	if err != nil {
		return "", err
	}
	return cid, nil
}
func startContainerByName(containerName string) (string, error) {
	cid, err := stopOrStartContainerByName(containerName, true)
	if err != nil {
		return "", err
	}
	return cid, nil
}

// StopNode stops the IPFS cluster node by stopping the Docker containers associated with the peer.
func (c *IpfsClusterPeer) StopNode() error {
	// Stop the containers by name
	if _, err := stopContainerByName(c.GetIpfsContainerName()); err != nil {
		return err
	}
	if _, err := stopContainerByName(c.GetClusterContainerName()); err != nil {
		return err
	}

	// Reset the container ID after stopping
	c.containerID = ""
	return nil
}

// StartNode starts the IPFS cluster node by launching the Docker containers associated with the peer.
func (c *IpfsClusterPeer) StartNode() error {
	// Launch the containers by generating and running the Docker Compose file
	if err := c.LaunchNode(); err != nil {
		return err
	}
	_, err := startContainerByName(c.GetIpfsContainerName())
	if err != nil {
		return err
	}
	// Set the container ID after starting
	cid, err := startContainerByName(c.GetClusterContainerName())
	if err != nil {
		return err
	}
	c.containerID = cid
	return nil
}

func (c *IpfsClusterPeer) GetFile(cid string) error {
	var ctlCmd *exec.Cmd
	if c.erasureEnabled {
		ctlCmd = exec.Command("docker", "exec", c.GetClusterContainerName(), "ipfs-cluster-ctl", "ecget", cid)
	} else {
		ctlCmd = exec.Command("docker", "exec", c.GetClusterContainerName(), "ipfs-cluster-ctl", "get", cid)
	}

	timeout := c.runenv.IntParam("getFileTimeout")
	out, err := executeCommandWithTimeout(ctlCmd, timeout, fmt.Errorf("timout retrieving file %s", cid))
	c.runenv.RecordMessage("\n\nOUTPUT\n\n%s", out)
	if err != nil {
		return err
	}

	_, e := exec.Command("docker", "exec", c.GetClusterContainerName(), "ls", "-la").CombinedOutput()
	if e != nil {
		return e
	}
	return nil
}

func (c *IpfsClusterPeer) ClearIPFSCache() error {
	cmd := exec.Command("docker", "exec", c.GetClusterContainerName(), "ipfs-cluster-ctl", "ipfs", "gc")
	_, err := executeCommandWithTimeout(cmd, 10, fmt.Errorf("clear IPFS Cache Timed Out"))
	return err
}

func (c *IpfsClusterPeer) PrintPinnedFiles(fileName string) error {
	cmd := exec.Command("docker", "exec", c.GetClusterContainerName(), "ipfs-cluster-ctl", "status", "--filter", "pinned")
	out, err := executeCommandWithTimeout(cmd, 15, fmt.Errorf("ipfs-cluster-ctl status --filter pinned: timed out"))
	c.runenv.RecordMessage("PrintPinnedFilesO")

	s, _ := os.Getwd()
	c.runenv.RecordMessage(s)
	// Write output to a file
	file, _ := os.Create(fileName)
	defer file.Close()

	file.Write([]byte(out))

	return err
}

// PinFile adds a file to the IPFS cluster using ipfs-cluster-ctl.
// The 'filePath' parameter specifies the path of the file to add.
func (c *IpfsClusterPeer) PinFile(filePath string) (*ECFile, error) {
	// Check if the file exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("file '%s' does not exist", filePath)
	}

	// Copy the file from the host to the IPFS container
	copyCmd := exec.Command("docker", "cp", filePath, fmt.Sprintf("%s:/data/", c.GetClusterContainerName()))
	if err := copyCmd.Run(); err != nil {
		return nil, fmt.Errorf("error copying file to IPFS container: %v", err)
	}

	// Execute ipfs-cluster-ctl add command inside the IPFS container\
	var ctlCmd *exec.Cmd
	if c.erasureEnabled {
		ctlCmd = exec.Command("docker", "exec", c.GetClusterContainerName(), "ipfs-cluster-ctl", "add", "/data/"+filepath.Base(filePath), "--erasure", "--shard-size", "512000")
	} else {
		ctlCmd = exec.Command("docker", "exec", c.GetClusterContainerName(), "ipfs-cluster-ctl", "add", "/data/"+filepath.Base(filePath))
	}

	// Wait for the operation to complete or timeout
	timeout := c.runenv.IntParam("fileUpsertTimeout")
	// Execute the command and capture output/errors
	// Signal that the operation is completed
	// Command execution completed
	// If an error occurred during command execution, return it
	// Timeout occurred
	out, err := executeCommandWithTimeout(ctlCmd, timeout, fmt.Errorf("timeout adding file to IPFS cluster: %s", filePath))
	if err != nil {
		return nil, err
	}
	c.runenv.RecordMessage(out)
	var ecFile *ECFile
	if c.erasureEnabled {
		ecFile = newECFileFromOutput(out)
	} else {
		ecFile = newFileFromErasureDisabledOutput(out)
	}

	return ecFile, nil
}

func executeCommand(ctlCmd *exec.Cmd) (string, error) {
	return executeCommandWithTimeout(ctlCmd, 60*10, fmt.Errorf("timed out"))
}
func executeCommandWithTimeout(ctlCmd *exec.Cmd, timeoutSeconds int, timeoutError error) (string, error) {
	// Channel to signal when the operation is completed
	var output []byte
	var cmdErr error
	done := make(chan struct{})
	go func() {
		defer close(done)

		output, cmdErr = ctlCmd.CombinedOutput()
		log.Printf("\n\nRAWOUTPUT:\n\n%s", output)
		done <- struct{}{}
	}()

	select {
	case <-done:
		if cmdErr != nil {
			return "", cmdErr
		}
		return string(output), nil
	case <-time.After(time.Duration(timeoutSeconds) * time.Second):

		return "", timeoutError
	}
}
