package session

import (
	"bufio"
	"fmt"
	"strings"
	"testing"
)

func responseItemJSON(role, text string) string {
	return fmt.Sprintf(
		`{"type":"response_item","payload":{"type":"message","role":"%s","content":[{"type":"input_text","text":"%s"}]}}`,
		role,
		text,
	)
}

func TestConversationHeadFromLine_MultilingualAndEmoji(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{name: "chinese", text: "请帮我实现会话恢复功能"},
		{name: "english", text: "please add retry logic for scanner"},
		{name: "spanish", text: "por favor corrige el analisis de sesiones"},
		{name: "latin", text: "salve quaeso sessiones refice"},
		{name: "japanese", text: "セッション一覧の表示を最適化してください"},
		{name: "korean", text: "세션 스캐너 성능을 개선해 주세요"},
		{name: "arabic", text: "يرجى تحسين فحص الجلسات"},
		{name: "mixed", text: "请修复 list command gracias 日本語も対応 مرحبا"},
		{name: "emoji-mixed", text: "fix flaky test 😄🔥 in scanner"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := conversationHeadFromLine([]byte(responseItemJSON("user", tc.text)))
			if got != tc.text {
				t.Fatalf("head mismatch\nwant: %q\ngot:  %q", tc.text, got)
			}
		})
	}
}

func TestReadConversationHead_LargeConversation(t *testing.T) {
	var b strings.Builder
	target := "请帮我实现多语言 session restore 支持吗？"

	for i := 0; i < 8; i++ {
		fmt.Fprintf(&b, "%s\n", responseItemJSON("user", fmt.Sprintf("status update %03d", i)))
	}
	fmt.Fprintf(&b, "%s\n", responseItemJSON("user", target))
	for i := 8; i < 280; i++ {
		fmt.Fprintf(&b, "%s\n", responseItemJSON("user", fmt.Sprintf("status update %03d", i)))
	}
	fmt.Fprintf(&b, "%s\n", responseItemJSON("assistant", "ack"))
	for i := 281; i < 420; i++ {
		fmt.Fprintf(&b, "%s\n", responseItemJSON("user", fmt.Sprintf("note %03d", i)))
	}

	head := readConversationHead(bufio.NewReader(strings.NewReader(b.String())))
	if head != target {
		t.Fatalf("unexpected head\nwant: %q\ngot:  %q", target, head)
	}
}

func TestReadConversationHead_SkipsOverlongLine(t *testing.T) {
	var b strings.Builder
	huge := strings.Repeat("x", maxSessionHeadLineBytes+128)
	target := "please keep this head after oversized line"

	fmt.Fprintf(&b, "%s\n", responseItemJSON("user", huge))
	fmt.Fprintf(&b, "%s\n", responseItemJSON("user", target))

	head := readConversationHead(bufio.NewReader(strings.NewReader(b.String())))
	if head != target {
		t.Fatalf("unexpected head after oversize line\nwant: %q\ngot:  %q", target, head)
	}
}
