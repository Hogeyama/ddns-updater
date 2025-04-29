package stun

import (
	"fmt"
	"net"
	"time"

	"github.com/pion/stun"
)

func GetIPv4AndAvailableTcpPort() (string, int, error) {
	const stunServer = "stunserver2025.stunprotocol.org:3478"

	remoteAddr, err := net.ResolveTCPAddr("tcp", stunServer)
	if err != nil {
		return "", 0, fmt.Errorf("failed to resolve STUN server address: %w", err)
	}

	// TCPでSTUNサーバに接続
	conn, err := net.DialTCP("tcp", nil, remoteAddr)
	if err != nil {
		return "", 0, fmt.Errorf("failed to connect to STUN server: %w", err)
	}
	defer conn.Close()

	// タイムアウト設定
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	// STUN Binding Request作成
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	// 送信
	if _, err = conn.Write(message.Raw); err != nil {
		return "", 0, fmt.Errorf("failed to send STUN request: %w", err)
	}

	// 受信
	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read STUN response: %w", err)
	}

	// レスポンス解析
	var response stun.Message
	response.Raw = buf[:n]
	if err := response.Decode(); err != nil {
		return "", 0, fmt.Errorf("failed to decode STUN response: %w", err)
	}

	// XOR-MAPPED-ADDRESS取得
	var xorAddr stun.XORMappedAddress
	if err := xorAddr.GetFrom(&response); err != nil {
		return "", 0, fmt.Errorf("failed to get XOR-MAPPED-ADDRESS: %w", err)
	}

	return xorAddr.IP.String(), xorAddr.Port, nil
}

// GetIPv4FromLocalPort discovers external IP and port using a specific local UDP port
func GetIPv4FromLocalPort(localPort int) (string, int, error) {
	const stunServer = "stunserver2025.stunprotocol.org:3478"

	// UDPでSTUNサーバに接続（指定されたローカルポートを使用）
	localAddr := &net.UDPAddr{Port: localPort}
	remoteAddr, err := net.ResolveUDPAddr("udp", stunServer)
	if err != nil {
		return "", 0, fmt.Errorf("failed to resolve STUN server address: %w", err)
	}

	conn, err := net.DialUDP("udp", localAddr, remoteAddr)
	if err != nil {
		return "", 0, fmt.Errorf("failed to connect to STUN server: %w", err)
	}
	defer conn.Close()

	// タイムアウト設定
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	// STUN Binding Request作成
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	// 送信
	if _, err = conn.Write(message.Raw); err != nil {
		return "", 0, fmt.Errorf("failed to send STUN request: %w", err)
	}

	// 受信
	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read STUN response: %w", err)
	}

	// レスポンス解析
	var response stun.Message
	response.Raw = buf[:n]
	if err := response.Decode(); err != nil {
		return "", 0, fmt.Errorf("failed to decode STUN response: %w", err)
	}

	// XOR-MAPPED-ADDRESS取得
	var xorAddr stun.XORMappedAddress
	if err := xorAddr.GetFrom(&response); err != nil {
		return "", 0, fmt.Errorf("failed to get XOR-MAPPED-ADDRESS: %w", err)
	}

	return xorAddr.IP.String(), xorAddr.Port, nil
}
