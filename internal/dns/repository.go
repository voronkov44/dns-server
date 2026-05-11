package dns

import (
	"context"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Repository interface {
	ListServers(ctx context.Context) ([]Server, error)
	AddServer(ctx context.Context, server Server) ([]Server, error)
	DeleteServer(ctx context.Context, server Server) ([]Server, error)
}

type FileRepository struct {
	path string
	mu   sync.Mutex
}

func NewFileRepository(path string) *FileRepository {
	return &FileRepository{
		path: path,
	}
}

func (r *FileRepository) ListServers(ctx context.Context) ([]Server, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// защита от одновременного выполнения внутри одного процесса сервера
	r.mu.Lock()
	defer r.mu.Unlock()

	content, err := r.readFile()
	if err != nil {
		return nil, err
	}

	return parseServersFromContent(string(content)), nil
}

func (r *FileRepository) AddServer(ctx context.Context, server Server) ([]Server, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	contentBytes, err := r.readFile()
	if err != nil {
		return nil, err
	}

	content := string(contentBytes)
	servers := parseServersFromContent(content)

	for _, existing := range servers {
		if sameAddress(existing.Address, server.Address) {
			return nil, ErrServerAlreadyExists
		}
	}

	line := "nameserver " + server.Address

	if strings.TrimSpace(content) == "" {
		content = line + "\n"
	} else {
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += line + "\n"
	}

	if err := r.writeFileAtomic([]byte(content)); err != nil {
		return nil, err
	}

	return parseServersFromContent(content), nil
}

func (r *FileRepository) DeleteServer(ctx context.Context, server Server) ([]Server, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	contentBytes, err := r.readFile()
	if err != nil {
		return nil, err
	}

	content := string(contentBytes)
	lines := strings.Split(content, "\n")

	filtered := make([]string, 0, len(lines))
	removed := false

	for _, line := range lines {
		fields := strings.Fields(line)

		if len(fields) >= 2 && fields[0] == "nameserver" && sameAddress(fields[1], server.Address) {
			removed = true
			continue
		}

		filtered = append(filtered, line)
	}

	if !removed {
		return nil, ErrServerNotFound
	}

	newContent := strings.Join(filtered, "\n")

	if strings.TrimSpace(newContent) != "" && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}

	if err := r.writeFileAtomic([]byte(newContent)); err != nil {
		return nil, err
	}

	return parseServersFromContent(newContent), nil
}

func (r *FileRepository) readFile() ([]byte, error) {
	if err := r.ensureFileExists(); err != nil {
		return nil, err
	}

	return os.ReadFile(r.path)
}

func (r *FileRepository) ensureFileExists() error {
	_, err := os.Stat(r.path)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	dir := filepath.Dir(r.path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	file, err := os.OpenFile(r.path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	return file.Close()
}

func (r *FileRepository) writeFileAtomic(content []byte) error {
	targetPath, err := r.resolvedPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(targetPath)
	base := filepath.Base(targetPath)

	mode := os.FileMode(0644)
	if info, err := os.Stat(targetPath); err == nil {
		mode = info.Mode().Perm()
	}

	tmpFile, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return err
	}

	tmpPath := tmpFile.Name()
	shouldRemoveTmp := true

	defer func() {
		if shouldRemoveTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(content); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if err := tmpFile.Chmod(mode); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		return err
	}

	shouldRemoveTmp = false

	if err := syncDir(dir); err != nil {
		return err
	}

	return nil
}

func (r *FileRepository) resolvedPath() (string, error) {
	if err := r.ensureFileExists(); err != nil {
		return "", err
	}

	resolved, err := filepath.EvalSymlinks(r.path)
	if err != nil {
		return r.path, nil
	}

	return resolved, nil
}

func syncDir(dir string) error {
	dirFile, err := os.Open(dir)
	if err != nil {
		return err
	}

	syncErr := dirFile.Sync()
	closeErr := dirFile.Close()

	if syncErr != nil {
		return syncErr
	}

	return closeErr
}

func parseServersFromContent(content string) []Server {
	servers := make([]Server, 0)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)

		if len(fields) < 2 || fields[0] != "nameserver" {
			continue
		}

		address, ok := normalizeAddress(fields[1])
		if !ok {
			continue
		}

		servers = append(servers, Server{
			Address: address,
		})
	}

	return servers
}

func normalizeAddress(address string) (string, bool) {
	address = strings.TrimSpace(address)

	parsed, err := netip.ParseAddr(address)
	if err != nil {
		return "", false
	}

	return parsed.String(), true
}

func sameAddress(left, right string) bool {
	normalizedLeft, okLeft := normalizeAddress(left)
	normalizedRight, okRight := normalizeAddress(right)

	if okLeft && okRight {
		return normalizedLeft == normalizedRight
	}

	return strings.TrimSpace(left) == strings.TrimSpace(right)
}
