package dataFromWS

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// minimal struct 用于示例解析（可选，原始数据仍在 messages）
type klineEvent struct {
	Event  string `json:"e"` // event type
	Symbol string `json:"s"`
	K      struct {
		StartTime int64  `json:"t"`
		Open      string `json:"o"`
		Close     string `json:"c"`
		High      string `json:"h"`
		Low       string `json:"l"`
		IsClosed  bool   `json:"x"`
	} `json:"k"`
}

func TestSubscribeKlines(t *testing.T) {
	// 15 秒内至少拿到一条消息（本地网络依赖）
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	msgCh, errCh, closeFn := SubscribeKlines(ctx, "ETHUSDT", "1m", "http://127.0.0.1:7897")
	defer closeFn()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("received error from websocket: %v", err)
		}
	case msg := <-msgCh:
		// 打印原始 JSON（测试输出）
		t.Logf("raw message: %s", string(msg))

		// 尝试解析成示例 struct（不强制）
		var ev klineEvent
		if err := json.Unmarshal(msg, &ev); err != nil {
			t.Logf("failed to unmarshal into klineEvent (ok): %v", err)
		} else {
			t.Logf("parsed event symbol=%s event=%s k.start=%d close=%s closed=%v",
				ev.Symbol, ev.Event, ev.K.StartTime, ev.K.Close, ev.K.IsClosed)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for message (check network / endpoint)")
	}
}
