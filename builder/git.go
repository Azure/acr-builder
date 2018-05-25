package builder

import (
	"bytes"
	"context"
	"os"
	"strings"
)

// GetGitCommitID queries git for the latest commit.
func (b *Builder) GetGitCommitID(ctx context.Context, cmdDir string) (string, error) {
	cmd := []string{"git", "rev-parse", "--verify", "HEAD"}
	var buf bytes.Buffer
	if err := b.cmder.Run(ctx, cmd, nil, &buf, os.Stderr, cmdDir); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}
