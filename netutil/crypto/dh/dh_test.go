package dh

import (
	"bytes"
	"crypto/rc4"
	"fmt"
	"math/big"
	"testing"
)

func TestDH(t *testing.T) {
	X1, E1 := DHExchange()
	X2, E2 := DHExchange()

	fmt.Println("Secret 1:", X1, E1)
	fmt.Println("Secret 2:", X2, E2)

	KEY1 := DHKey(X1, E2)
	KEY2 := DHKey(X2, E1)

	fmt.Println("KEY1:", KEY1)
	fmt.Println("KEY2:", KEY2)

	if KEY1.Cmp(KEY2) != 0 {
		t.Error("Diffie-Hellman failed")
	}
}

const (
	DecodeSaleSvr = 1430886687
	EncodeSaleSvr = 1430886783
)

const (
	DecodeSaleClient = 1430886687
	EncodeSaleClient = 1430886783
)

//模拟服务器客户端交换密钥
func TestDHExchange(t *testing.T) {
	recvKeyClient, recvSeedClient := DHExchange()
	sendKeyClient, sendSeedClient := DHExchange()

	recvKeySvr, recvSeedSvr := DHExchange()
	sendKeySvr, sendSeedSvr := DHExchange()

	encodeSvrKey := DHKey(sendKeySvr, big.NewInt(recvSeedClient.Int64()))
	decodeClientKey := DHKey(recvKeySvr, big.NewInt(sendSeedClient.Int64()))
	t.Logf("server encode key:%v,decode key:%v", encodeSvrKey, decodeClientKey)
	encoderSvr, _ := rc4.NewCipher([]byte(fmt.Sprintf("%v%v", DecodeSaleSvr, decodeClientKey))) //[]byte(fmt.Sprintf("%v", decodeClientKey))
	decoderSvr, _ := rc4.NewCipher([]byte(fmt.Sprintf("%v%v", EncodeSaleSvr, encodeSvrKey)))    //[]byte(fmt.Sprintf("%v", encodeSvrKey))

	decodeSvrKey := DHKey(sendKeyClient, big.NewInt(recvSeedSvr.Int64()))
	encodeClientKey := DHKey(recvKeyClient, big.NewInt(sendSeedSvr.Int64()))
	t.Logf("client encode key:%v,decode key:%v", encodeClientKey, decodeSvrKey)
	encoderClient, _ := rc4.NewCipher([]byte(fmt.Sprintf("%v%v", EncodeSaleClient, encodeClientKey))) //[]byte(fmt.Sprintf("%v", encodeClientKey))
	decoderClient, _ := rc4.NewCipher([]byte(fmt.Sprintf("%v%v", DecodeSaleClient, decodeSvrKey)))    //[]byte(fmt.Sprintf("%v", decodeSvrKey))

	//服务器加密发送，客户端解密读取
	sendSvrStr := []byte("server send msg")
	sendSvrStrEnr := make([]byte, len(sendSvrStr))
	encoderSvr.XORKeyStream(sendSvrStrEnr, sendSvrStr)

	recvSvrStr := make([]byte, len(sendSvrStrEnr))
	decoderClient.XORKeyStream(recvSvrStr, sendSvrStrEnr)
	if !bytes.Equal(sendSvrStr, recvSvrStr) {
		t.Errorf("client decrypt err")
	}

	//客户端加密发送，服务器解密读取
	sendClientStr := []byte("client send msg")
	sendClientStrEnr := make([]byte, len(sendClientStr))
	encoderClient.XORKeyStream(sendClientStrEnr, sendClientStr)

	recvClientStr := make([]byte, len(sendClientStrEnr))
	decoderSvr.XORKeyStream(recvClientStr, sendClientStrEnr)
	if !bytes.Equal(sendClientStr, recvClientStr) {
		t.Errorf("server decrypt err")
	}

}

func BenchmarkDH(b *testing.B) {
	for i := 0; i < b.N; i++ {
		X1, E1 := DHExchange()
		X2, E2 := DHExchange()

		DHKey(X1, E2)
		DHKey(X2, E1)
	}
}
