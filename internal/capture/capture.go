// Package capture 监听裸 TCP，原样记录 HTTP 请求的字节。
//
// 这里不用 net/http，因为标准库会把 header 规范化（首字母大写、排序、去重、丢重复项），
// 那样就拿不到原始顺序和大小写了。我们自己读 socket 字节，自己找 header 边界，
// 自己按 Content-Length / Transfer-Encoding: chunked 读 body。
package capture

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/yuebai-blast/raw-lens/internal/store"
)

// Serve 在 addr 上监听并把抓到的请求写进 st。
//
// tlsConf 非 nil 时把 listener 包一层 TLS：握手后 tls.Conn 仍是 net.Conn，
// 从它读到的是解密后的明文字节——正好是客户端发的原始 HTTP 请求，下游逻辑无需改动。
func Serve(addr string, st *store.Store, tlsConf *tls.Config) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	scheme := "http"
	if tlsConf != nil {
		ln = tls.NewListener(ln, tlsConf)
		scheme = "https"
	}
	log.Printf("capture 监听 %s (%s)", addr, scheme)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept 失败: %v", err)
			continue
		}
		go handleConn(conn, st)
	}
}

func handleConn(conn net.Conn, st *store.Store) {
	defer conn.Close()
	// 给整条连接一个读超时，避免半开连接挂住 goroutine。
	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	br := bufio.NewReader(conn)
	raw, headerBlock, body, err := readRawRequest(br)
	if len(raw) == 0 {
		return // 空连接 / TLS 握手失败，忽略
	}
	cr := parseCaptured(raw, headerBlock, body, conn.RemoteAddr().String())
	_, cr.TLS = conn.(*tls.Conn)
	id := st.Add(cr)
	if err != nil {
		log.Printf("#%d 来自 %s（读取未完整: %v，已按收到的字节保存）", id, cr.RemoteAddr, err)
	} else {
		log.Printf("#%d %s %s 来自 %s（%d 字节）", id, cr.Method, cr.Target, cr.RemoteAddr, len(raw))
	}
	writeAck(conn, id)
}

// readRawRequest 读一条完整请求的原始字节。
// 返回 raw（header+body 全量）、headerBlock（含结尾空行）、body。
func readRawRequest(br *bufio.Reader) (raw, headerBlock, body []byte, err error) {
	headerBlock, err = readUntilHeaderEnd(br)
	if err != nil {
		return headerBlock, headerBlock, nil, err
	}
	n, chunked := bodyLength(headerBlock)
	switch {
	case chunked:
		body, err = readChunked(br)
	case n > 0:
		body = make([]byte, n)
		_, err = io.ReadFull(br, body)
	}
	raw = make([]byte, 0, len(headerBlock)+len(body))
	raw = append(raw, headerBlock...)
	raw = append(raw, body...)
	return raw, headerBlock, body, err
}

const maxHeaderBytes = 1 << 20 // 1 MiB header 上限，防滥用

// readUntilHeaderEnd 一直读到 header 结束（\r\n\r\n，也容忍裸 \n\n）。
func readUntilHeaderEnd(br *bufio.Reader) ([]byte, error) {
	var buf []byte
	for {
		b, err := br.ReadByte()
		if err != nil {
			return buf, err
		}
		buf = append(buf, b)
		n := len(buf)
		if n >= 4 && buf[n-1] == '\n' && buf[n-2] == '\r' && buf[n-3] == '\n' && buf[n-4] == '\r' {
			return buf, nil
		}
		if n >= 2 && buf[n-1] == '\n' && buf[n-2] == '\n' {
			return buf, nil
		}
		if n > maxHeaderBytes {
			return buf, fmt.Errorf("header 超过 %d 字节", maxHeaderBytes)
		}
	}
}

// bodyLength 从 header 块里读出 body 长度信息（大小写不敏感地匹配 header 名）。
func bodyLength(headerBlock []byte) (n int, chunked bool) {
	for _, line := range splitLines(headerBlock) {
		idx := bytes.IndexByte(line, ':')
		if idx < 0 {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(string(line[:idx])))
		val := strings.TrimSpace(string(line[idx+1:]))
		switch name {
		case "content-length":
			if v, e := strconv.Atoi(val); e == nil && v >= 0 {
				n = v
			}
		case "transfer-encoding":
			if strings.Contains(strings.ToLower(val), "chunked") {
				chunked = true
			}
		}
	}
	return n, chunked
}

// readChunked 原样读完 chunked body（连同分块框架字节一起返回，保真）。
func readChunked(br *bufio.Reader) ([]byte, error) {
	var buf []byte
	for {
		line, err := br.ReadBytes('\n')
		buf = append(buf, line...)
		if err != nil {
			return buf, err
		}
		sizeField := strings.TrimSpace(string(line))
		if i := strings.IndexByte(sizeField, ';'); i >= 0 { // chunk extension
			sizeField = sizeField[:i]
		}
		size, err := strconv.ParseInt(strings.TrimSpace(sizeField), 16, 64)
		if err != nil {
			return buf, fmt.Errorf("非法 chunk 大小 %q: %w", sizeField, err)
		}
		if size == 0 {
			// 末块，读 trailer 直到空行
			for {
				tl, err := br.ReadBytes('\n')
				buf = append(buf, tl...)
				if err != nil {
					return buf, err
				}
				if len(strings.TrimSpace(string(tl))) == 0 {
					return buf, nil
				}
			}
		}
		data := make([]byte, size+2) // 数据 + 结尾 CRLF
		_, err = io.ReadFull(br, data)
		buf = append(buf, data...)
		if err != nil {
			return buf, err
		}
	}
}

// splitLines 按行切（同时容忍 \r\n 和裸 \n），去掉结尾空行。
func splitLines(block []byte) [][]byte {
	parts := bytes.Split(block, []byte("\n"))
	out := make([][]byte, 0, len(parts))
	for _, p := range parts {
		out = append(out, bytes.TrimSuffix(p, []byte("\r")))
	}
	for len(out) > 0 && len(out[len(out)-1]) == 0 {
		out = out[:len(out)-1]
	}
	return out
}

// parseCaptured 把原始字节解析成结构化字段，header 的顺序和大小写完全保留。
func parseCaptured(raw, headerBlock, body []byte, remote string) *store.CapturedRequest {
	cr := &store.CapturedRequest{
		Time:       time.Now(),
		RemoteAddr: remote,
		Raw:        raw,
		Body:       body,
	}
	lines := splitLines(headerBlock)
	if len(lines) == 0 {
		return cr
	}
	cr.RequestLine = string(lines[0])
	if f := strings.SplitN(cr.RequestLine, " ", 3); len(f) > 0 {
		cr.Method = f[0]
		if len(f) > 1 {
			cr.Target = f[1]
		}
		if len(f) > 2 {
			cr.Proto = f[2]
		}
	}
	for _, line := range lines[1:] {
		if len(line) == 0 {
			continue
		}
		idx := bytes.IndexByte(line, ':')
		if idx < 0 {
			cr.Headers = append(cr.Headers, [2]string{string(line), ""})
			continue
		}
		name := string(line[:idx])
		// 只去掉冒号后惯例的一个前导空格，其余保持原样。
		val := strings.TrimPrefix(string(line[idx+1:]), " ")
		cr.Headers = append(cr.Headers, [2]string{name, val})
	}
	return cr
}

// writeAck 回一个最小的 200 响应。用 Connection: close，一条连接处理一条请求。
func writeAck(conn net.Conn, id int64) {
	body := fmt.Sprintf("raw-lens captured request #%d\n", id)
	fmt.Fprintf(conn,
		"HTTP/1.1 200 OK\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\nContent-Length: %d\r\n\r\n%s",
		len(body), body)
}
