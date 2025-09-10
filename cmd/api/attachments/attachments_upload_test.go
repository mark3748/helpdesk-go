package attachments

import (
	"bytes"
	"github.com/gin-gonic/gin"
	apppkg "github.com/mark3748/helpdesk-go/cmd/api/app"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUploadObject_InvalidKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true, MinIOBucket: "attachments", ObjectStoreTimeoutMS: 500}
	a := apppkg.NewApp(cfg, nil, nil, &apppkg.FsObjectStore{Base: dir}, nil)

	h := UploadObject(a)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	req := httptest.NewRequest(http.MethodPut, "/attachments/upload/not-a-uuid", bytes.NewReader([]byte("hello")))
	c.Request = req
	c.Params = gin.Params{{Key: "objectKey", Value: "not-a-uuid"}}
	h(c)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestUploadObject_Filesystem_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	cfg := apppkg.Config{Env: "test", TestBypassAuth: true, MinIOBucket: "attachments", ObjectStoreTimeoutMS: 500}
	a := apppkg.NewApp(cfg, nil, nil, &apppkg.FsObjectStore{Base: dir}, nil)

	key := "123e4567-e89b-12d3-a456-426614174000"
	h := UploadObject(a)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	req := httptest.NewRequest(http.MethodPut, "/attachments/upload/"+key, bytes.NewReader([]byte("hello")))
	req.Header.Set("Content-Type", "text/plain")
	c.Request = req
	c.Params = gin.Params{{Key: "objectKey", Value: key}}
	h(c)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}
