package mocks

import (
	"net"
	"time"
)

type MockConn struct {
	ReadData  []byte
	WriteData []byte
	Closed    bool
	Err       error
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

func (m *MockConn) Close() error {
	m.Closed = true
	return m.Err
}

func (m *MockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{}
}

func (m *MockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{}
}

func (m *MockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}
