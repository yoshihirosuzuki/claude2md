package render

import "io"

// errWriter は io.Writer をラップし、最初に発生したエラーを記憶して以降の Write を no-op にする。
// これにより、本パッケージ内のレンダリング関数は各 Write のエラーを毎回チェックせず、
// 最後に 1 度だけ ew.err を確認すれば足りる。
type errWriter struct {
	w   io.Writer
	err error
}

func newErrWriter(w io.Writer) *errWriter {
	return &errWriter{w: w}
}

func (e *errWriter) Write(p []byte) (int, error) {
	if e.err != nil {
		return 0, e.err
	}
	n, err := e.w.Write(p)
	if err != nil {
		e.err = err
	}
	return n, err
}
