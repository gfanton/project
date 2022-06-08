package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
)

type GitClient struct{}

func Git() *GitClient {
	return &GitClient{}
}

func (g *GitClient) CloneContext(ctx context.Context, provider, path, dest string) error {
	url := g.getUrl(provider, path)
	cmd := exec.CommandContext(ctx, "git", "clone", url, dest)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	return cmd.Run()
}

func (g *GitClient) getUrl(provider, path string) string {
	if g.hasSShAgent() {
		return fmt.Sprintf("git@%s:%s", provider, path)
	}

	return fmt.Sprintf("https://%s/%s.git", provider, path)
}

func (g *GitClient) hasSShAgent() bool {
	s := os.Getenv("SSH_AUTH_SOCK")
	return s != ""
}

// func getAgentClient(user string) (transport.Transport, error) {
// 	cmd := exec.Command("tr", "a-z", "A-Z")
// 	// ssh-agent(1) provides a UNIX socket at $SSH_AUTH_SOCK.
// 	socket := os.Getenv("SSH_AUTH_SOCK")
// 	conn, err := net.Dial("unix", socket)
// 	if err != nil {
// 		return nil, fmt.Errorf("Failed to open SSH_AUTH_SOCK: %w", err)
// 	}

// 	if user == "" {
// 		if user, err = username(); err != nil {
// 			return nil, fmt.Errorf("unable to get username: %w", err)
// 		}
// 	}

// 	agentClient := agent.NewClient(conn)
// 	config := &ssh.ClientConfig{
// 		User: user,
// 		Auth: []ssh.AuthMethod{
// 			// Use a callback rather than PublicKeys so we only consult the
// 			// agent once the remote server wants it.
// 			ssh.PublicKeysCallback(agentClient.Signers),
// 		},
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 	}

// 	// agent, err := git_ssh.NewSSHAgentAuth("")
// 	// if err != nil {
// 	// 	return fmt.Errorf("unable to get ssh agent: %w", err)
// 	// }

// 	client := git_ssh.NewClient(config)

// 	git_ssh
// 	return client, nil
// }

func username() (string, error) {
	var username string
	if user, err := user.Current(); err == nil {
		username = user.Username
	} else {
		username = os.Getenv("USER")
	}

	if username == "" {
		return "", errors.New("failed to get username")
	}

	return username, nil
}

// func hasSSHAgent() bool {
// 	auth_socket := os.Getenv("SSH_AUTH_SOCK")
// }

// const (
// 	Username    = "sszuecs"
// 	DefaultPort = 22
// )

// var Ch chan string = make(chan string)

// func KeyScanCallback(hostname string, remote net.Addr, key ssh.PublicKey) error {
// 	Ch <- fmt.Sprintf("%s %s", hostname[:len(hostname)-3], string(ssh.MarshalAuthorizedKey(key)))
// 	return nil
// }

// func dial(server string, config *ssh.ClientConfig, wg *sync.WaitGroup) {
// 	_, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", server, DefaultPort), config)
// 	if err != nil {
// 		log.Fatalln("Failed to dial:", err)
// 	}
// 	wg.Done()

// }

// func out(wg *sync.WaitGroup) {
// 	for s := range Ch {
// 		fmt.Printf("%s", s)
// 		wg.Done()
// 	}
// }

// func main() {
// 	auth_socket := os.Getenv("SSH_AUTH_SOCK")
// 	if auth_socket == "" {
// 		log.Fatal(errors.New("no $SSH_AUTH_SOCK defined"))
// 	}
// 	conn, err := net.Dial("unix", auth_socket)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer conn.Close()
// 	ag := agent.NewClient(conn)
// 	auths := []ssh.AuthMethod{ssh.PublicKeysCallback(ag.Signers)}

// 	config := &ssh.ClientConfig{
// 		User:            Username,
// 		Auth:            auths,
// 		HostKeyCallback: KeyScanCallback,
// 	}

// 	var wg sync.WaitGroup
// 	go out(&wg)
// 	reader := bufio.NewReader(os.Stdin)
// 	for {
// 		server, err := reader.ReadString('\n')
// 		if err == io.EOF {
// 			break
// 		}
// 		server = server[:len(server)-1] // chomp
// 		wg.Add(2)                       // dial and print
// 		go dial(server, config, &wg)
// 	}
// 	wg.Wait()
// }
