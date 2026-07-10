package rpc

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockNode returns an RPCClient pointed at a server that always replies with the
// given JSON-RPC body.
func mockNode(t *testing.T, body string) *RPCClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewRPCClient("test", srv.URL, "5s")
}

// A node reply with a null result must return an error, not panic on the nil
// *json.RawMessage dereference (an unrecovered panic would kill the pool).
func TestNullResultReturnsErrorNotPanic(t *testing.T) {
	const nullResult = `{"jsonrpc":"2.0","id":0,"result":null}`

	if _, err := mockNode(t, nullResult).GetWork(); err == nil {
		t.Error("GetWork: expected an error for a null result")
	}
	if _, err := mockNode(t, nullResult).SubmitBlock([]string{"0x0", "0x0", "0x0"}); err == nil {
		t.Error("SubmitBlock: expected an error for a null result")
	}
	if _, err := mockNode(t, nullResult).GetBalance("0x0"); err == nil {
		t.Error("GetBalance: expected an error for a null result")
	}
	if _, err := mockNode(t, nullResult).GetPeerCount(); err == nil {
		t.Error("GetPeerCount: expected an error for a null result")
	}
	if _, err := mockNode(t, nullResult).SendTransaction("0x0", "0x1", "", "", "0x0", true); err == nil {
		t.Error("SendTransaction: expected an error for a null result")
	}
}

// An error object without a string "message" field must not panic the type
// assertion in doPost.
func TestErrorWithoutMessageReturnsErrorNotPanic(t *testing.T) {
	c := mockNode(t, `{"jsonrpc":"2.0","id":0,"error":{"code":-1}}`)
	if _, err := c.GetWork(); err == nil {
		t.Error("expected an error for a node error reply without a message")
	}
}

// A whole-body JSON null response is valid JSON that decodes to a nil struct;
// it must return an error, not panic dereferencing it in doPost.
func TestNullBodyReturnsErrorNotPanic(t *testing.T) {
	c := mockNode(t, `null`)
	if _, err := c.GetWork(); err == nil {
		t.Fatal("expected an error for a null response body")
	}
}

func TestGetWorkValid(t *testing.T) {
	c := mockNode(t, `{"jsonrpc":"2.0","id":0,"result":["0xheader","0xseed","0xtarget"]}`)
	work, err := c.GetWork()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(work) != 3 || work[0] != "0xheader" {
		t.Fatalf("unexpected work: %v", work)
	}
}
