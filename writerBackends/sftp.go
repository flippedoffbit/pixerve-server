package writerbackends

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"pixerve/logger"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// UploadToSFTPWithCreds uploads content from an io.Reader to a remote server via SFTP.
// accessInfo should contain at least: host, user, remotePath. Optionally: port (default 22), password or privateKey (base64 or raw PEM).
func UploadToSFTPWithCreds(ctx context.Context, accessInfo map[string]string, reader io.Reader) error {
	host := accessInfo["host"]
	port := accessInfo["port"]
	if port == "" {
		port = "22"
	}
	user := accessInfo["user"]
	password := accessInfo["password"]
	privateKey := accessInfo["privateKey"]
	remotePath := accessInfo["remotePath"]

	if host == "" || user == "" || remotePath == "" {
		return fmt.Errorf("missing required accessInfo keys: host, user, remotePath")
	}

	var auths []ssh.AuthMethod
	if privateKey != "" {
		// try to decode as base64, fall back to raw
		keyBytes, err := base64.StdEncoding.DecodeString(privateKey)
		if err != nil {
			keyBytes = []byte(privateKey)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return fmt.Errorf("parse private key: %w", err)
		}
		auths = append(auths, ssh.PublicKeys(signer))
	} else if password != "" {
		auths = append(auths, ssh.Password(password))
	} else {
		return fmt.Errorf("no auth method provided; set password or privateKey in accessInfo")
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := net.JoinHostPort(host, port)

	// Dial respecting context
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial tcp %s: %w", addr, err)
	}

	// perform SSH handshake on the established connection
	clientConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return fmt.Errorf("ssh handshake with %s: %w", addr, err)
	}
	sshClient := ssh.NewClient(clientConn, chans, reqs)
	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("create sftp client: %w", err)
	}
	defer sftpClient.Close()

	// Ensure remote directory exists
	dir := path.Dir(remotePath)
	if err := mkdirAllSFTP(sftpClient, dir); err != nil {
		return fmt.Errorf("ensure remote dir %s: %w", dir, err)
	}

	// Create (or truncate) remote file and copy data
	f, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote file %s: %w", remotePath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return fmt.Errorf("copy to remote file %s: %w", remotePath, err)
	}

	logger.Infof("Successfully uploaded '%s' to %s", remotePath, addr)
	return nil
}

// mkdirAllSFTP mimics os.MkdirAll for an SFTP server by creating each segment of the path.
func mkdirAllSFTP(client *sftp.Client, dir string) error {
	if dir == "" || dir == "." || dir == "/" {
		return nil
	}

	// Normalize and split path - use strings since sftp paths are posix-like
	parts := strings.Split(dir, "/")
	cur := ""
	if strings.HasPrefix(dir, "/") {
		cur = "/"
	}

	for _, p := range parts {
		if p == "" {
			continue
		}
		cur = path.Join(cur, p)
		if _, err := client.Stat(cur); err != nil {
			if os.IsNotExist(err) {
				if err := client.Mkdir(cur); err != nil {
					return fmt.Errorf("mkdir %s: %w", cur, err)
				}
			} else {
				return fmt.Errorf("stat %s: %w", cur, err)
			}
		}
	}
	return nil
}

func UseUploadToSFTPWithCredsExample() {
	// Example values - do NOT hardcode credentials in production.
	accessInfo := map[string]string{
		"host":       "sftp.example.com",
		"port":       "22",
		"user":       "username",
		"password":   "secret",
		"remotePath": "/upload/example.txt",
	}

	content := "This is a test upload to SFTP."
	reader := strings.NewReader(content)

	if err := UploadToSFTPWithCreds(context.TODO(), accessInfo, reader); err != nil {
		logger.Fatal(err)
	}
}
