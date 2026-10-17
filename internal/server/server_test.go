package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	systemv1 "github.com/wimpheling/heph-v2/gen/heph/system/v1"
	"github.com/wimpheling/heph-v2/gen/heph/system/v1/systemv1connect"
)

func TestSystemInfoJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, systemv1connect.SystemServiceGetSystemInfoProcedure, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	res := httptest.NewRecorder()
	NewHandler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", res.Code, res.Body)
	}
	var info struct {
		Name    string
		Version string
	}
	if err := json.Unmarshal(res.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if info.Name != "heph-v2" || info.Version != "0.1.0" {
		t.Fatalf("unexpected system info: %+v", info)
	}
}

func TestUnknownRPC(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/heph.system.v1.SystemService/Missing", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	res := httptest.NewRecorder()
	NewHandler().ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.Code)
	}
}

func TestProtocolsAndShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- serve(ctx, listener, slog.New(slog.NewTextHandler(io.Discard, nil))) }()

	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	client := &http.Client{Transport: &http.Transport{Protocols: protocols}, Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	for _, tc := range []struct {
		name    string
		options []connect.ClientOption
	}{
		{name: "Connect protobuf"},
		{name: "gRPC", options: []connect.ClientOption{connect.WithGRPC()}},
		{name: "gRPC-Web", options: []connect.ClientOption{connect.WithGRPCWeb()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rpc := systemv1connect.NewSystemServiceClient(client, "http://"+listener.Addr().String(), tc.options...)
			requestCtx, stop := context.WithTimeout(ctx, 5*time.Second)
			defer stop()
			response, err := rpc.GetSystemInfo(requestCtx, connect.NewRequest(&systemv1.GetSystemInfoRequest{}))
			if err != nil {
				t.Fatal(err)
			}
			if response.Msg.GetName() != "heph-v2" || response.Msg.GetVersion() != "0.1.0" {
				t.Fatalf("unexpected system info: %v", response.Msg)
			}
		})
	}
	client.CloseIdleConnections()
	cancel()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("server did not stop after cancellation")
	}
	connection, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err == nil {
		connection.Close()
		t.Fatal("listener is still accepting connections")
	}
}
