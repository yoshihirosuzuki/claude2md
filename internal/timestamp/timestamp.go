// Package timestamp は Claude.ai エクスポート JSON 中の RFC3339(Nano) 文字列を
// パースする処理を集約する。export スキーマのフォーマットが将来変わった場合に、
// 修正箇所が 1 ヶ所で済むようにするための薄いラッパー。
package timestamp

import (
	"fmt"
	"time"
)

// Parse は RFC3339Nano（小数秒は任意桁）として s をパースする。
// Claude.ai エクスポートの created_at / updated_at は通常マイクロ秒 6 桁の RFC3339Nano。
func Parse(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q: %w", s, err)
	}
	return t, nil
}

// Compare は a と b の前後を返す。a > b なら +1、a < b なら -1、等しければ 0。
// パース失敗時は error を返す。
func Compare(a, b string) (int, error) {
	ta, err := Parse(a)
	if err != nil {
		return 0, err
	}
	tb, err := Parse(b)
	if err != nil {
		return 0, err
	}
	switch {
	case ta.After(tb):
		return 1, nil
	case ta.Before(tb):
		return -1, nil
	}
	return 0, nil
}
