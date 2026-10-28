package multiplayer

// WebSocket mínimo (RFC 6455) con la librería estándar: handshake, tramas de texto y cierre.
// Solo lo necesario para la demo; en producción usa una librería probada.

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type WSConn struct {
	c    net.Conn
	r    *bufio.Reader
	mask bool // los clientes enmascaran sus tramas; el servidor no
	wmu  sync.Mutex
}

func acceptKey(k string) string {
	h := sha1.Sum([]byte(k + wsGUID))
	return base64.StdEncoding.EncodeToString(h[:])
}

// Upgrade convierte una petición HTTP en una conexión WebSocket (lado servidor).
func Upgrade(w http.ResponseWriter, r *http.Request) (*WSConn, error) {
	k := r.Header.Get("Sec-WebSocket-Key")
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") || k == "" {
		return nil, errors.New("no es un handshake de WebSocket")
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("el servidor no permite hijack")
	}
	c, rw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", acceptKey(k))
	if err := rw.Flush(); err != nil {
		c.Close()
		return nil, err
	}
	return &WSConn{c: c, r: rw.Reader}, nil
}

// Dial abre una conexión WebSocket (lado cliente) a una URL ws://.
func Dial(raw string) (*WSConn, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	c, err := net.Dial("tcp", u.Host)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, 16)
	rand.Read(nonce)
	k := base64.StdEncoding.EncodeToString(nonce)
	path := u.RequestURI()
	fmt.Fprintf(c, "GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n", path, u.Host, k)
	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodGet})
	if err != nil {
		c.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols || resp.Header.Get("Sec-WebSocket-Accept") != acceptKey(k) {
		c.Close()
		return nil, fmt.Errorf("handshake rechazado: %s", resp.Status)
	}
	return &WSConn{c: c, r: br, mask: true}, nil
}

func (w *WSConn) writeFrame(op byte, p []byte) error {
	w.wmu.Lock()
	defer w.wmu.Unlock()
	hdr := []byte{0x80 | op}
	var mbit byte
	if w.mask {
		mbit = 0x80
	}
	switch n := len(p); {
	case n < 126:
		hdr = append(hdr, mbit|byte(n))
	case n < 1<<16:
		hdr = append(hdr, mbit|126)
		hdr = binary.BigEndian.AppendUint16(hdr, uint16(n))
	default:
		hdr = append(hdr, mbit|127)
		hdr = binary.BigEndian.AppendUint64(hdr, uint64(n))
	}
	if w.mask {
		var mk [4]byte
		rand.Read(mk[:])
		hdr = append(hdr, mk[:]...)
		q := make([]byte, len(p))
		for i := range p {
			q[i] = p[i] ^ mk[i%4]
		}
		p = q
	}
	_, err := w.c.Write(append(hdr, p...))
	return err
}

// ReadText devuelve el siguiente mensaje de texto completo; io.EOF si el otro lado cerró.
func (w *WSConn) ReadText() ([]byte, error) {
	var msg []byte
	for {
		var h [2]byte
		if _, err := io.ReadFull(w.r, h[:]); err != nil {
			return nil, err
		}
		fin, op, masked, n := h[0]&0x80 != 0, h[0]&0x0f, h[1]&0x80 != 0, uint64(h[1]&0x7f)
		switch n {
		case 126:
			var b [2]byte
			if _, err := io.ReadFull(w.r, b[:]); err != nil {
				return nil, err
			}
			n = uint64(binary.BigEndian.Uint16(b[:]))
		case 127:
			var b [8]byte
			if _, err := io.ReadFull(w.r, b[:]); err != nil {
				return nil, err
			}
			n = binary.BigEndian.Uint64(b[:])
		}
		var mk [4]byte
		if masked {
			if _, err := io.ReadFull(w.r, mk[:]); err != nil {
				return nil, err
			}
		}
		p := make([]byte, n)
		if _, err := io.ReadFull(w.r, p); err != nil {
			return nil, err
		}
		if masked {
			for i := range p {
				p[i] ^= mk[i%4]
			}
		}
		switch op {
		case 0x8:
			return nil, io.EOF
		case 0x9:
			_ = w.writeFrame(0xA, p) // ping → pong
			continue
		case 0xA:
			continue
		}
		msg = append(msg, p...)
		if fin {
			return msg, nil
		}
	}
}

func (w *WSConn) WriteMsg(m Msg) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return w.writeFrame(0x1, b)
}

func (w *WSConn) ReadMsg() (Msg, error) {
	var m Msg
	b, err := w.ReadText()
	if err != nil {
		return m, err
	}
	return m, json.Unmarshal(b, &m)
}

// Close manda la trama de cierre y suelta la conexión.
func (w *WSConn) Close() error {
	_ = w.writeFrame(0x8, nil)
	return w.c.Close()
}
