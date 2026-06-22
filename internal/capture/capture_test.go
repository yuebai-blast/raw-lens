package capture

import (
	"bufio"
	"io"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// parse 跑的是 handleConn 里真正用的那条链路：readRawRequest 读字节、parseCaptured 解析。
// 不开 socket，直接喂字节，专门盯「字节保真」这个不变量。
func parse(t *testing.T, raw string) (rawBytes, body []byte, cr structured) {
	t.Helper()
	br := bufio.NewReader(strings.NewReader(raw))
	rb, headerBlock, b, err := readRawRequest(br)
	// 良构请求（Content-Length 准确或 chunked 完整）读完不应报错；
	// 容忍 io.EOF：无 body 时底层可能已读到流尾。
	if err != nil && err != io.EOF {
		t.Fatalf("readRawRequest 出错: %v", err)
	}
	c := parseCaptured(rb, headerBlock, b, "127.0.0.1:1234")
	return rb, b, structured{c.RequestLine, c.Method, c.Target, c.Proto, c.Headers}
}

// structured 只取 parseCaptured 解析出的结构化字段，方便断言。
type structured struct {
	RequestLine string
	Method      string
	Target      string
	Proto       string
	Headers     [][2]string
}

// 核心不变量：Raw 必须与客户端发来的字节逐字节一致，绝不做任何规范化。
func TestRawBytesAreVerbatim(t *testing.T) {
	// 故意构造「非规范」的请求：乱序、大小写混杂、重复 header、冒号后空格不一。
	input := "GET /a/b?x=1 HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"x-LoWeR-AnD-uPpEr: KeepMe\r\n" +
		"X-Dup: first\r\n" +
		"X-Dup: second\r\n" +
		"Accept:no-space\r\n" +
		"X-Pad:   three-leading\r\n" +
		"Content-Length: 11\r\n" +
		"\r\n" +
		"hello world"

	rawBytes, body, _ := parse(t, input)
	if string(rawBytes) != input {
		t.Errorf("Raw 不是逐字节保真:\n  want %q\n  got  %q", input, string(rawBytes))
	}
	if string(body) != "hello world" {
		t.Errorf("body = %q, want %q", string(body), "hello world")
	}
}

// header 的顺序、大小写、重复项都必须原样保留——这正是不走 net/http 的理由
// （标准库会首字母大写、排序、合并重复项）。
func TestHeaderOrderCaseDuplicatesPreserved(t *testing.T) {
	input := "GET / HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"x-LoWeR-AnD-uPpEr: KeepMe\r\n" +
		"X-Dup: first\r\n" +
		"X-Dup: second\r\n" +
		"Accept:no-space\r\n" +
		"X-Pad:   three-leading\r\n" +
		"\r\n"

	_, _, cr := parse(t, input)
	want := [][2]string{
		{"Host", "example.com"},
		{"x-LoWeR-AnD-uPpEr", "KeepMe"},
		{"X-Dup", "first"},
		{"X-Dup", "second"},
		{"Accept", "no-space"},
		{"X-Pad", "  three-leading"}, // 冒号后三个空格，只去掉一个
	}
	if !reflect.DeepEqual(cr.Headers, want) {
		t.Errorf("Headers 不匹配:\n  want %v\n  got  %v", want, cr.Headers)
	}
}

// header 值只去掉冒号后惯例的「一个」前导空格，其余（含 tab、尾随空格、多余空格）原样保留。
func TestHeaderValueOnlyStripsOneLeadingSpace(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"X: v", "v"},             // 一个前导空格被去掉
		{"X:v", "v"},              // 没有前导空格，原样
		{"X:  v", " v"},           // 两个空格，只去一个
		{"X:\tv", "\tv"},          // tab 不是空格，保留
		{"X: v ", "v "},           // 尾随空格保留
		{"X: ", ""},               // 仅一个空格，去掉后为空
		{"X:   ", "  "},           // 三个空格，去一个剩两个
		{"X: a: b: c", "a: b: c"}, // 只在第一个冒号处切分
	}
	for _, c := range cases {
		input := "GET / HTTP/1.1\r\n" + c.line + "\r\n\r\n"
		_, _, cr := parse(t, input)
		if len(cr.Headers) != 1 {
			t.Fatalf("%q: 期望解析出 1 个 header，得到 %v", c.line, cr.Headers)
		}
		if cr.Headers[0][1] != c.want {
			t.Errorf("%q: 值 = %q, want %q", c.line, cr.Headers[0][1], c.want)
		}
	}
}

// 没有冒号的 header 行整行作为名、值为空，保留而不丢弃。
func TestHeaderLineWithoutColon(t *testing.T) {
	input := "GET / HTTP/1.1\r\nHost: a\r\nGarbageLineNoColon\r\n\r\n"
	_, _, cr := parse(t, input)
	want := [][2]string{
		{"Host", "a"},
		{"GarbageLineNoColon", ""},
	}
	if !reflect.DeepEqual(cr.Headers, want) {
		t.Errorf("Headers = %v, want %v", cr.Headers, want)
	}
}

// 请求行拆成 method/target/proto，原始 target（含 query、奇异字符）原样保留。
func TestRequestLineParsing(t *testing.T) {
	cases := []struct {
		line                  string
		method, target, proto string
	}{
		{"GET / HTTP/1.1", "GET", "/", "HTTP/1.1"},
		{"POST /a/b?x=1&y=2 HTTP/1.0", "POST", "/a/b?x=1&y=2", "HTTP/1.0"},
		{"CONNECT example.com:443 HTTP/1.1", "CONNECT", "example.com:443", "HTTP/1.1"},
		{"WeIrDmEtHoD /p HTTP/2", "WeIrDmEtHoD", "/p", "HTTP/2"},
	}
	for _, c := range cases {
		input := c.line + "\r\nHost: a\r\n\r\n"
		_, _, cr := parse(t, input)
		if cr.RequestLine != c.line {
			t.Errorf("RequestLine = %q, want %q", cr.RequestLine, c.line)
		}
		if cr.Method != c.method || cr.Target != c.target || cr.Proto != c.proto {
			t.Errorf("%q -> method=%q target=%q proto=%q; want %q/%q/%q",
				c.line, cr.Method, cr.Target, cr.Proto, c.method, c.target, c.proto)
		}
	}
}

// bodyLength 大小写不敏感地识别 Content-Length 与 Transfer-Encoding: chunked。
func TestBodyLengthCaseInsensitive(t *testing.T) {
	cases := []struct {
		header      string
		wantN       int
		wantChunked bool
	}{
		{"Content-Length: 5", 5, false},
		{"content-length: 5", 5, false},
		{"CONTENT-LENGTH: 5", 5, false},
		{"Content-Length:   42", 42, false}, // 值前后空格不影响
		{"Transfer-Encoding: chunked", 0, true},
		{"Transfer-Encoding: Chunked", 0, true},
		{"transfer-encoding: CHUNKED", 0, true},
		{"Transfer-Encoding: gzip, chunked", 0, true}, // 含 chunked 即可
		{"X-Whatever: 5", 0, false},                   // 无关 header
	}
	for _, c := range cases {
		block := []byte("GET / HTTP/1.1\r\n" + c.header + "\r\n\r\n")
		n, chunked := bodyLength(block)
		if n != c.wantN || chunked != c.wantChunked {
			t.Errorf("%q -> n=%d chunked=%v; want n=%d chunked=%v",
				c.header, n, chunked, c.wantN, c.wantChunked)
		}
	}
}

// Content-Length 指定的 body 精确按字节读，Raw = header + body 且逐字节一致。
func TestContentLengthBodyExact(t *testing.T) {
	// body 里故意含 CRLF 和会被误判为 header 边界的字节，确认按长度读而非按分隔符。
	bodyContent := "a\r\nb\r\n\r\nc"
	input := "POST / HTTP/1.1\r\nContent-Length: " +
		strconv.Itoa(len(bodyContent)) + "\r\n\r\n" + bodyContent
	rawBytes, body, _ := parse(t, input)
	if string(body) != bodyContent {
		t.Errorf("body = %q, want %q", string(body), bodyContent)
	}
	if string(rawBytes) != input {
		t.Errorf("Raw 非保真:\n  want %q\n  got  %q", input, string(rawBytes))
	}
}

// chunked body 连同分块框架字节（大小行、CRLF、末块、trailer）一起原样保存。
func TestChunkedFramingPreserved(t *testing.T) {
	chunkBody := "5\r\nhello\r\n" + // 第一块
		"6\r\n world\r\n" + // 第二块
		"0\r\n" + // 末块大小
		"X-Trailer: t\r\n" + // trailer
		"\r\n" // 结束空行
	input := "POST / HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n" + chunkBody

	rawBytes, body, _ := parse(t, input)
	if string(body) != chunkBody {
		t.Errorf("chunked body 未原样保存:\n  want %q\n  got  %q", chunkBody, string(body))
	}
	if string(rawBytes) != input {
		t.Errorf("Raw 非保真:\n  want %q\n  got  %q", input, string(rawBytes))
	}
}

// chunk extension（大小后跟 ;name=value）也要原样保留在 body 里。
func TestChunkedExtensionPreserved(t *testing.T) {
	chunkBody := "5;ext=1\r\nhello\r\n0\r\n\r\n"
	input := "POST / HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n" + chunkBody
	_, body, _ := parse(t, input)
	if string(body) != chunkBody {
		t.Errorf("chunk extension 未保真:\n  want %q\n  got  %q", chunkBody, string(body))
	}
}

// header 结尾容忍裸 \n\n（不是 \r\n\r\n），且 Raw 原样。
func TestBareLFHeaderTermination(t *testing.T) {
	input := "GET / HTTP/1.1\nHost: a\nX-Bare: b\n\n"
	rawBytes, _, cr := parse(t, input)
	if string(rawBytes) != input {
		t.Errorf("Raw 非保真:\n  want %q\n  got  %q", input, string(rawBytes))
	}
	want := [][2]string{{"Host", "a"}, {"X-Bare", "b"}}
	if !reflect.DeepEqual(cr.Headers, want) {
		t.Errorf("Headers = %v, want %v", cr.Headers, want)
	}
}

// readUntilHeaderEnd 在 CRLFCRLF 处恰好停下，不多读 body 一个字节。
func TestReadUntilHeaderEndStopsAtBoundary(t *testing.T) {
	header := "GET / HTTP/1.1\r\nHost: a\r\n\r\n"
	br := bufio.NewReader(strings.NewReader(header + "BODYBYTES"))
	block, err := readUntilHeaderEnd(br)
	if err != nil {
		t.Fatalf("readUntilHeaderEnd: %v", err)
	}
	if string(block) != header {
		t.Errorf("headerBlock = %q, want %q", string(block), header)
	}
	rest, _ := io.ReadAll(br)
	if string(rest) != "BODYBYTES" {
		t.Errorf("边界后剩余字节 = %q, want %q", string(rest), "BODYBYTES")
	}
}

// header 超过 1 MiB 上限时报错（防滥用），但已收到的字节仍返回。
func TestHeaderSizeLimit(t *testing.T) {
	huge := "GET / HTTP/1.1\r\nX-Big: " + strings.Repeat("A", maxHeaderBytes+10) + "\r\n\r\n"
	br := bufio.NewReader(strings.NewReader(huge))
	block, err := readUntilHeaderEnd(br)
	if err == nil {
		t.Fatal("超长 header 期望报错，实际无错误")
	}
	if len(block) <= maxHeaderBytes {
		t.Errorf("返回字节数 = %d，期望 > %d（已收到的字节应保留）", len(block), maxHeaderBytes)
	}
}
