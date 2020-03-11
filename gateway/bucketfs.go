package gateway

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/textileio/textile/api/buckets/client"
	"github.com/textileio/textile/api/common"
)

type serveBucketFileSystem interface {
	Exists(bucket, pth string) bool
	Write(bucket, pth string, writer io.Writer) error
	ValidHost() string
}

type bucketFileSystem struct {
	client  *client.Client
	token   string
	timeout time.Duration
	host    string
}

// serveBucket returns a middleware handler that serves files in a bucket.
func serveBucket(fs serveBucketFileSystem) gin.HandlerFunc {
	return func(c *gin.Context) {
		bucket, err := bucketFromHost(c.Request.Host, fs.ValidHost())
		if err != nil {
			return
		}
		if fs.Exists(bucket, c.Request.URL.Path) {
			c.Writer.WriteHeader(http.StatusOK)
			ctype := mime.TypeByExtension(filepath.Ext(c.Request.URL.Path))
			if ctype == "" {
				ctype = "application/octet-stream"
			}
			c.Writer.Header().Set("Content-Type", ctype)
			if err := fs.Write(bucket, c.Request.URL.Path, c.Writer); err != nil {
				abort(c, http.StatusInternalServerError, err)
			} else {
				c.Abort()
			}
		}
	}
}

func (f *bucketFileSystem) Exists(bucket, pth string) bool {
	if bucket == "" || pth == "/" {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()
	rep, err := f.client.ListPath(common.NewDevTokenContext(ctx, f.token), path.Join(bucket, pth))
	if err == nil && !rep.Item.IsDir {
		return true
	}
	return false
}

func (f *bucketFileSystem) Write(bucket, pth string, writer io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()
	return f.client.PullPath(common.NewDevTokenContext(ctx, f.token), path.Join(bucket, pth), writer)
}

func (f *bucketFileSystem) ValidHost() string {
	return f.host
}

func bucketFromHost(host, valid string) (buck string, err error) {
	parts := strings.SplitN(host, ".", 2)
	if parts[len(parts)-1] != valid {
		err = fmt.Errorf("invalid bucket host")
		return
	}
	return parts[0], nil
}
