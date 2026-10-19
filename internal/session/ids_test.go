package session

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// imagePartSig 计算 part 原文摘要（sha256 前 8 hex），供期望值精确构造——
// 签名算法变更时测试期望值随此 helper 单点同步。
func imagePartSig(t *testing.T, part string) string {
	t.Helper()
	sum := sha256.Sum256([]byte(part))
	return hex.EncodeToString(sum[:4])
}

// TestResolveConversationID 覆盖 conversationId 提取的 snake/camel/缺失三态：
//   - metadata.conversation_id / metadata.conversationId → 取值
//   - 顶层 conversation_id / conversationId → 取值（snake 优先同 ExtractKey）
//   - 缺失 / 只有 user_id → ""（会话头族语义只认对话 ID，绝不回落 user_id）
func TestResolveConversationID(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"metadata snake_case", `{"metadata":{"conversation_id":"conv-1"}}`, "conv-1"},
		{"metadata camelCase", `{"metadata":{"conversationId":"conv-2"}}`, "conv-2"},
		{"top-level snake_case", `{"conversation_id":"conv-3"}`, "conv-3"},
		{"top-level camelCase", `{"conversationId":"conv-4"}`, "conv-4"},
		{"snake wins over camel", `{"conversation_id":"conv-s","conversationId":"conv-c"}`, "conv-s"},
		{"missing", `{"model":"glm-5.2"}`, ""},
		{"empty body", ``, ""},
		{"broken json", `{broken`, ""},
		{"metadata user_id only", `{"metadata":{"user_id":"u1"}}`, ""},
		{"top-level user_id only", `{"user_id":"u1"}`, ""},
		{"empty string value", `{"conversationId":""}`, ""},
		{"non-string value", `{"conversationId":123}`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ResolveConversationID([]byte(c.body)); got != c.want {
				t.Errorf("ResolveConversationID(%q) = %q want %q", c.body, got, c.want)
			}
		})
	}
}

func TestResolveConversationIDForRequest(t *testing.T) {
	firstTurn := []byte(`{"messages":[{"role":"user","content":"help me debug this service"}]}`)
	secondTurn := []byte(`{"messages":[{"role":"user","content":"help me debug this service"},{"role":"assistant","content":"Sure"},{"role":"user","content":"inspect the logs"}]}`)

	fromSession := ResolveConversationIDForRequest(firstTurn, "", "agent:main:thread:123")
	if fromSession == "" || !isUUIDv5(fromSession) {
		t.Fatalf("session-derived conversation ID = %q; want UUIDv5", fromSession)
	}
	if strings.Contains(fromSession, "agent:main:thread:123") {
		t.Errorf("session identifier leaked in conversation ID: %q", fromSession)
	}
	if got := ResolveConversationIDForRequest(firstTurn, "", "agent:main:thread:123"); got != fromSession {
		t.Errorf("same session must yield stable ID: %q != %q", got, fromSession)
	}
	if got := ResolveConversationIDForRequest(firstTurn, "", "agent:main:thread:456"); got == fromSession {
		t.Errorf("different sessions must yield different IDs: %q", got)
	}

	fromFirstMessage := ResolveConversationIDForRequest(firstTurn, "", "")
	if got := ResolveConversationIDForRequest(secondTurn, "", ""); got != fromFirstMessage {
		t.Errorf("same first user message must keep a stable ID across turns: %q != %q", got, fromFirstMessage)
	}
	if got := ResolveConversationIDForRequest([]byte(`{"prompt_cache_key":"cache-1","messages":[]}`), "", ""); got != conversationUUIDv5("body-key:cache-1") {
		t.Errorf("body cache key fallback = %q", got)
	}
	for _, body := range []string{
		`{"user":"global-user","messages":[{"role":"user","content":"same session opener"}]}`,
		`{"user_id":"global-user","messages":[{"role":"user","content":"same session opener"}]}`,
		`{"metadata":{"user_id":"global-user"},"messages":[{"role":"user","content":"same session opener"}]}`,
	} {
		if got := ResolveConversationIDForRequest([]byte(body), "", ""); got == "" || got != conversationUUIDv5("first-user:same session opener") {
			t.Errorf("user-scoped request should not collapse conversation IDs to user_id: got %q for body %s", got, body)
		}
	}
	if got := ResolveConversationIDForRequest([]byte(`{"conversationId":"body-conv"}`), "header-conv", "session-1"); got != "body-conv" {
		t.Errorf("explicit body ID must win: %q", got)
	}
	if got := ResolveConversationIDForRequest(nil, "header-conv", "session-1"); got != "header-conv" {
		t.Errorf("explicit conversation header must win: %q", got)
	}
	if got := ResolveConversationIDForRequest(nil, "", "session-1"); got != conversationUUIDv5("session:session-1") {
		t.Errorf("session header fallback = %q", got)
	}
	if got := ResolveConversationIDForRequest([]byte(`{"messages":[]}`), "", ""); got != "" {
		t.Errorf("request with no session signal must not fabricate an ID: %q", got)
	}
}

func TestConversationUUIDv5ReferenceVectors(t *testing.T) {
	// RFC 4122 name-based UUID reference vectors with the process salt fixed to a
	// known byte sequence. This checks the algorithm, not just its UUID shape.
	previousSalt := deriveSalt
	deriveSalt = "00112233445566778899aabbccddeeff"
	t.Cleanup(func() { deriveSalt = previousSalt })
	cases := []struct {
		name string
		want string
	}{
		{"session:abc", "2ca830de-ec16-5d1c-85c6-2d33dc9f749e"},
		{"first-user:hi", "02336081-4842-5a11-a25d-7fcdbddc31b4"},
	}
	for _, tc := range cases {
		if got := conversationUUIDv5(tc.name); got != tc.want {
			t.Errorf("conversationUUIDv5(%q) = %q want %q", tc.name, got, tc.want)
		}
	}
}

func isUUIDv5(id string) bool {
	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' || id[14] != '5' || id[19] < '8' || id[19] > 'b' {
		return false
	}
	for i, c := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}

// TestNewMessageIDFormat 32 位 hex 且不为空、两次调用大概率不同（随机性冒烟）。
func TestNewMessageIDFormat(t *testing.T) {
	for i := 0; i < 50; i++ {
		id := NewMessageID()
		if len(id) != 32 {
			t.Fatalf("NewMessageID() = %q len=%d want 32", id, len(id))
		}
		for _, ch := range id {
			if !strings.ContainsRune("0123456789abcdef", ch) {
				t.Fatalf("NewMessageID() = %q has non-hex char %q", id, ch)
			}
		}
	}
}

// TestRequestIDForKeyDerivation 同 key 恒稳定、异 key 各不同、空 key 每次新值
// （纯派生实现，无包级缓存，测试间天然隔离）。
func TestRequestIDForKeyDerivation(t *testing.T) {
	k1a := RequestIDForKey("conv-a")
	k1b := RequestIDForKey("conv-a")
	if k1a != k1b {
		t.Errorf("same key should be stable: %q vs %q", k1a, k1b)
	}
	k2 := RequestIDForKey("conv-b")
	if k1a == k2 {
		t.Errorf("different keys should differ: %q", k1a)
	}
	// 空 key：每次调用生成新值（无会话则无"会话内稳定"语义）。
	e1 := RequestIDForKey("")
	e2 := RequestIDForKey("")
	if e1 == e2 {
		t.Errorf("empty key should yield fresh values each call: %q", e1)
	}
	// 稳定值自身也须是 32 hex（可作 B3 TraceId 直接使用）。
	for _, id := range []string{k1a, k2, e1} {
		if len(id) != 32 {
			t.Errorf("RequestIDForKey value %q len=%d want 32", id, len(id))
		}
	}
}

// TestTurnKeyExtraction 轮级兜底键的提取：取**最后一条** user 消息的「序号+文本」，
// 轮内追加 assistant/tool 消息不改变键；无 user / 无文本 / 坏 JSON 一律空串。
func TestTurnKeyExtraction(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"single user", `{"messages":[{"role":"user","content":"你好"}]}`, "u0:你好"},
		{"last user wins", `{"messages":[{"role":"user","content":"第一问"},{"role":"assistant","content":"答"},{"role":"user","content":"第二问"}]}`, "u2:第二问"},
		// agent 多步：轮内追加 assistant/tool 消息，末条 user 位置与内容不变 → 同键。
		{"agent step keeps same key", `{"messages":[{"role":"user","content":"任务"},{"role":"assistant","tool_calls":[{"id":"c1"}]},{"role":"tool","content":"结果"}]}`, "u0:任务"},
		{"multimodal parts text joined", `{"messages":[{"role":"user","content":[{"type":"text","text":"看图"},{"type":"image_url","image_url":{"url":"data:x"}}]}]}`, "u0:看图\n[image_url:" + imagePartSig(t, `{"type":"image_url","image_url":{"url":"data:x"}}`) + "]"},
		{"no user message", `{"messages":[{"role":"system","content":"sys"}]}`, ""},
		{"empty messages", `{"messages":[]}`, ""},
		{"messages key absent", `{"model":"glm-5.2"}`, ""},
		{"broken json", `{broken`, ""},
		{"empty body", ``, ""},
		{"empty content", `{"messages":[{"role":"user","content":""}]}`, ""},
		{"null content", `{"messages":[{"role":"user","content":null}]}`, ""},
		// 纯图片 content 现按内容签名派生非空轮级键（G1 修复，原为 ""）；
		// 键含 image part 摘要（sha256 前 8 hex）。
		{"image only content", `{"messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"x"}}]}]}`, "u0:[image_url:" + imagePartSig(t, `{"type":"image_url","image_url":{"url":"x"}}`) + "]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := TurnKey([]byte(c.body)); got != c.want {
				t.Errorf("TurnKey(%q) = %q want %q", c.body, got, c.want)
			}
		})
	}
}

// TestTurnRequestIDDerivation 轮级 ID 是纯派生：同键恒同值、异键异值、空键每次新值、
// 形态恒 32 hex。
func TestTurnRequestIDDerivation(t *testing.T) {
	a1 := TurnRequestID("u0:同一个问题")
	a2 := TurnRequestID("u0:同一个问题")
	if a1 != a2 {
		t.Errorf("same turn key should derive same id: %q vs %q", a1, a2)
	}
	if b := TurnRequestID("u0:另一个问题"); b == a1 {
		t.Errorf("different turn keys should derive different ids: %q", b)
	}
	// 同文本但序号不同（不同轮里内容相同的提问）也要分开。
	if c := TurnRequestID("u2:同一个问题"); c == a1 {
		t.Errorf("same text at different position should differ: %q", c)
	}
	// 空键：无轮可聚合 → 每次新值（保持原有请求级独立行为）。
	if e1, e2 := TurnRequestID(""), TurnRequestID(""); e1 == e2 {
		t.Errorf("empty turn key should yield fresh values each call: %q", e1)
	}
	for _, id := range []string{a1, TurnRequestID(""), TurnRequestID("x")} {
		if len(id) != 32 {
			t.Errorf("TurnRequestID value %q len=%d want 32", id, len(id))
		}
	}
}
