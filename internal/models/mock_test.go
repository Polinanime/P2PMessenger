package models_test

import (
	"net"
	"time"
)

type MockConn struct {
    ReadData  []byte
    WriteData []byte
}

func (m *MockConn) Read(b []byte) (n int, err error) {
    copy(b, m.ReadData)
    return len(m.ReadData), nil
}

func (m *MockConn) Write(b []byte) (n int, err error) {
    m.WriteData = make([]byte, len(b))
    copy(m.WriteData, b)
    return len(b), nil
}

func (m *MockConn) Close() error                       { return nil }
func (m *MockConn) LocalAddr() net.Addr               { return nil }
func (m *MockConn) RemoteAddr() net.Addr              { return nil }
func (m *MockConn) SetDeadline(t time.Time) error     { return nil }
func (m *MockConn) SetReadDeadline(t time.Time) error { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }
