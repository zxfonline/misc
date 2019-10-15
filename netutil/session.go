package netutil

import (
	"crypto/rc4"
	"encoding/binary"
	"io"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zxfonline/misc/golangtrace"

	"github.com/zxfonline/misc/chanutil"

	"github.com/zxfonline/misc/expvar"
	"github.com/zxfonline/misc/log"
	"github.com/zxfonline/misc/timefix"
	"github.com/zxfonline/misc/trace"
)

var _sessionID int64

const (
	PACKET_LIMIT = 65536
)

const (
	//消息报头字节数
	HEAD_SIZE = 4
	//消息号占用的字节数
	MSG_ID_SIZE = 2
)

const (
	SESS_KEYEXCG = 0x1 // 是否已经交换完毕KEY
	SESS_ENCRYPT = 0x2 // 是否可以开始加密
)

type NetPacket struct {
	MsgId   uint16
	Data    []byte
	Session *TCPSession
}

type NetConnIF interface {
	SetReadDeadline(t time.Time) error
	Close() error
	SetWriteDeadline(t time.Time) error
	Write(b []byte) (n int, err error)
	RemoteAddr() net.Addr
	Read(p []byte) (n int, err error)
}

type TCPSession struct {
	//*net.TCPConn
	Conn     NetConnIF
	SendChan chan *NetPacket
	ReadChan chan *NetPacket
	//离线消息管道,用于外部接收连接断开的消息并处理后续
	OffChan chan int64
	// ID
	SessionId int64
	// 会话标记
	Flag int32

	// 发送缓冲
	sendCache []byte
	//发送缓冲大小
	sendCacheSize uint32

	readDelay time.Duration
	sendDelay time.Duration
	// Declares how many times we will try to resend message
	MaxSendRetries int
	//发送管道满后是否需要关闭连接
	sendFullClose bool
	CloseState    chanutil.DoneChan
	maxRecvSize   uint32
	// 包频率包数
	rpmLimit uint32
	// 包频率检测间隔
	rpmInterval int64
	// 超过频率控制离线通知包
	offLineMsg *NetPacket

	OnLineTime  int64
	OffLineTime int64

	EncodeKey []byte
	DecodeKey []byte
	tr        golangtrace.Trace
	sendLock  sync.Mutex
}

//filter:true 过滤成功，抛弃该报文；false:过滤失败，继续执行该报文消息
func (s *TCPSession) HandleConn(filter func(*NetPacket) bool) {
	go s.ReadLoop(filter)
	go s.SendLoop()
}

//网络连接远程ip
func (s *TCPSession) RemoteAddr() net.Addr {
	return s.Conn.RemoteAddr()
}

//网络连接远程ip
func (s *TCPSession) RemoteIp() string {
	addr := s.Conn.RemoteAddr().String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	return net.ParseIP(host).String()
}

func (s *TCPSession) Send(packet *NetPacket) bool {
	if packet == nil {
		return false
	}
	if !s.sendFullClose { //阻塞发送，直到管道关闭
		select {
		case s.SendChan <- packet:
			if wait := len(s.SendChan); wait > cap(s.SendChan)/10*5 && wait%20 == 0 {
				log.Warnf("session send process,waitChan:%d/%d,msg:%d,session:%d,remote:%s", wait, cap(s.SendChan), packet.MsgId, s.SessionId, s.RemoteAddr())
			}
			return true
		case <-s.CloseState:
			return false
		}
	} else { //缓存管道满了会关闭连接
		select {
		case <-s.CloseState:
			return false
		case s.SendChan <- packet:
			if wait := len(s.SendChan); wait > cap(s.SendChan)/10*5 && wait%20 == 0 {
				log.Warnf("session send process,waitChan:%d/%d,msg:%d,session:%d,remote:%s", wait, cap(s.SendChan), packet.MsgId, s.SessionId, s.RemoteAddr())
			}
			return true
		default:
			log.Errorf("session sender overflow,close session,waitChan:%d,msg:%d,session:%d,remote:%s", len(s.SendChan), packet.MsgId, s.SessionId, s.RemoteAddr())
			s.Close()
			return false
		}
	}
}

// RC4加密解密
func (s *TCPSession) SetCipher(encodeKey, decodeKey []byte) error {
	if len(encodeKey) < 1 || len(encodeKey) > 256 {
		return rc4.KeySizeError(len(encodeKey))
	}

	if len(decodeKey) < 1 || len(decodeKey) > 256 {
		return rc4.KeySizeError(len(decodeKey))
	}

	s.EncodeKey = encodeKey
	s.DecodeKey = decodeKey
	s.Flag |= SESS_KEYEXCG
	return nil
}

//use for defer recover
func PrintPanicStack() {
	if x := recover(); x != nil {
		buf := make([]byte, 4<<20) // 4 KB should be enough
		n := runtime.Stack(buf, false)
		log.Errorf("Recovered %v\nStack:%s", x, buf[:n])
	}
}

func (s *TCPSession) ReadLoop(filter func(*NetPacket) bool) {
	defer PrintPanicStack()

	// 关闭发送
	defer s.Close()

	var delayTimer *time.Timer

	if s.readDelay > 0 {
		delayTimer = time.NewTimer(s.readDelay)
	}

	rpmStart := time.Now().Unix()
	rpmCount := uint32(0)

	//rpmMsgCount := 0

	// 4字节包长度
	header := make([]byte, HEAD_SIZE)
	for {
		// 读取超时
		if s.readDelay > 0 {
			s.Conn.SetReadDeadline(time.Now().Add(s.readDelay))
		}

		// 4字节包长度
		n, err := io.ReadFull(s.Conn, header)
		if err != nil {
			//if err != io.EOF {
			//	log.Warnf("error receiving header,bytes:%d,session:%d,remote:%s,err:%v", n, s.SessionId, s.RemoteAddr(), err)
			//}
			return
		}

		// packet data
		size := binary.BigEndian.Uint32(header)
		if s.maxRecvSize != 0 && size > s.maxRecvSize {
			log.Warnf("error receiving,size:%d,session:%d,remote:%s", size, s.SessionId, s.RemoteAddr())
			return
		}

		data := make([]byte, size)
		n, err = io.ReadFull(s.Conn, data)
		if err != nil || size < MSG_ID_SIZE {
			log.Warnf("error receiving body,bytes:%d,size:%d,session:%d,remote:%s,err:%v", n, size, s.SessionId, s.RemoteAddr(), err)
			return
		}
		// 收包频率控制
		if s.rpmLimit > 0 {
			rpmCount++

			// 达到限制包数
			if rpmCount > s.rpmLimit {
				now := time.Now().Unix()
				// 检测时间间隔
				if now-rpmStart < s.rpmInterval {
					// 提示操作太频繁三次后踢下线
					//rpmMsgCount++
					//if rpmMsgCount > 3 {
					// 发送频率过高的消息包
					s.Send(s.offLineMsg)
					log.Errorf("session rpm too high,%d/%d qps,session:%d,remote:%s", rpmCount, s.rpmInterval, s.SessionId, s.RemoteAddr())
					return
					//}
					//// 发送频率过高的消息包
					//s.Send(s.offLineMsg)
				}

				// 未超过限制
				rpmCount = 0
				rpmStart = now
			}
		}

		// 解密
		if s.Flag&SESS_ENCRYPT != 0 {
			decoder, _ := rc4.NewCipher(s.DecodeKey)
			decoder.XORKeyStream(data, data)
		}

		msgId := binary.BigEndian.Uint16(data[:MSG_ID_SIZE])

		pack := &NetPacket{msgId, data[MSG_ID_SIZE:], s}

		if s.readDelay > 0 {
			delayTimer.Reset(s.readDelay)
			if filter == nil {
				select {
				case s.ReadChan <- pack:
				case <-delayTimer.C:
					log.Warnf("session read busy or closed,waitChan:%d,session:%d,remote:%s", len(s.ReadChan), s.SessionId, s.RemoteAddr())
					return
				}
			} else {
				if ok := filter(pack); !ok {
					select {
					case s.ReadChan <- pack:
					case <-delayTimer.C:
						log.Warnf("session read busy or closed,waitChan:%d,session:%d,remote:%s", len(s.ReadChan), s.SessionId, s.RemoteAddr())
						return
					}
				}
			}
		} else {
			if filter == nil {
				s.ReadChan <- pack
			} else {
				if ok := filter(pack); !ok {
					s.ReadChan <- pack
				}
			}
		}
	}
}

func (s *TCPSession) Close() {
	s.CloseState.SetDone()
}

func (s *TCPSession) IsClosed() bool {
	return s.CloseState.R().Done()
}

func (s *TCPSession) closeTask() {
	s.OffLineTime = timefix.SecondTime()
	// 告诉服务器该玩家掉线,或者下线
	t := time.NewTimer(15 * time.Second)
	select {
	case s.OffChan <- s.SessionId:
		t.Stop()
	case <-t.C:
		log.Warnf("off chan time out,session:%d,remote:%s", s.SessionId, s.RemoteAddr())
	}
	// close connection
	s.Conn.Close()
}
func (s *TCPSession) SendLoop() {
	defer PrintPanicStack()

	for {
		select {
		case <-s.CloseState:
			s.closeTask()
			return
		case packet := <-s.SendChan:
			s.DirectSend(packet)
		}
	}
}

func (s *TCPSession) DirectSend(packet *NetPacket) bool {
	if packet == nil {
		return true
	}
	if s.IsClosed() {
		return false
	}
	s.sendLock.Lock()
	defer s.sendLock.Unlock()

	packLen := uint32(len(packet.Data) + MSG_ID_SIZE)

	if packLen > s.sendCacheSize {
		s.sendCacheSize = packLen + (packLen >> 2) //1.25倍率
		s.sendCache = make([]byte, s.sendCacheSize)
	}

	// 4字节包长度
	binary.BigEndian.PutUint32(s.sendCache, packLen)

	// 2字节消息id
	binary.BigEndian.PutUint16(s.sendCache[HEAD_SIZE:], packet.MsgId)

	copy(s.sendCache[HEAD_SIZE+MSG_ID_SIZE:], packet.Data)

	// encryption
	// (NOT_ENCRYPTED) -> KEYEXCG -> ENCRYPT
	if s.Flag&SESS_ENCRYPT != 0 { // encryption is enabled
		encoder, _ := rc4.NewCipher(s.EncodeKey)
		data := s.sendCache[HEAD_SIZE : HEAD_SIZE+packLen]

		encoder.XORKeyStream(data, data)
	} else if s.Flag&SESS_KEYEXCG != 0 { // key is exchanged, encryption is not yet enabled
		s.Flag &^= SESS_KEYEXCG
		s.Flag |= SESS_ENCRYPT
	}

	err := s.performSend(s.sendCache[:HEAD_SIZE+packLen], 0)
	if err != nil {
		log.Warnf("error writing msg,session:%d,remote:%s,err:%v", s.SessionId, s.RemoteAddr(), err)
		s.Close()
		return false
	}
	return true
}

func (s *TCPSession) performSend(data []byte, sendRetries int) error {
	// 写超时
	if s.sendDelay > 0 {
		s.Conn.SetWriteDeadline(time.Now().Add(s.sendDelay))
	}

	_, err := s.Conn.Write(data)
	if err != nil {
		return s.processSendError(err, data, sendRetries)
	}
	return nil
}

func (s *TCPSession) processSendError(err error, data []byte, sendRetries int) error {
	netErr, ok := err.(net.Error)
	if !ok {
		return err
	}

	if s.isNeedToResendMessage(netErr, sendRetries) {
		return s.performSend(data, sendRetries+1)
	}
	//if !netErr.Temporary() {
	//	//重连,让外部来重连吧
	//}
	return err
}

func (s *TCPSession) isNeedToResendMessage(err net.Error, sendRetries int) bool {
	return (err.Temporary() || err.Timeout()) && sendRetries < s.MaxSendRetries
}

// 设置链接参数
func (s *TCPSession) SetParameter(readDelay, sendDelay time.Duration, maxRecvSize uint32, sendFullClose bool) {
	s.maxRecvSize = maxRecvSize
	if readDelay >= 0 {
		s.readDelay = readDelay
	}
	if sendDelay >= 0 {
		s.sendDelay = sendDelay
	}
	s.sendFullClose = sendFullClose
}

// 包频率控制参数
func (s *TCPSession) SetRpmParameter(rpmLimit uint32, rpmInterval int64, msg *NetPacket) {
	s.rpmLimit = rpmLimit
	s.rpmInterval = rpmInterval
	s.offLineMsg = msg
}

func NewSession(conn NetConnIF, readChan, sendChan chan *NetPacket, offChan chan int64) *TCPSession {
	s := &TCPSession{
		Conn:          conn,
		SendChan:      sendChan,
		ReadChan:      readChan,
		OffChan:       offChan,
		SessionId:     atomic.AddInt64(&_sessionID, 1),
		sendCache:     make([]byte, PACKET_LIMIT),
		sendCacheSize: PACKET_LIMIT,
		readDelay:     30 * time.Second,
		sendDelay:     10 * time.Second,
		sendFullClose: true,
		maxRecvSize:   10 * 1024,
		OnLineTime:    timefix.SecondTime(),
		CloseState:    chanutil.NewDoneChan(),
	}
	return s
}

func (s *TCPSession) ParsePacket(pkt []byte) *NetPacket {
	defer PrintPanicStack()
	if len(pkt) <= HEAD_SIZE {
		log.Warnf("error parse packet,bytes:%d,session:%d,remote:%s", len(pkt), s.SessionId, s.RemoteAddr())
		return nil
	}
	// 4字节包长度
	// packet data
	size := binary.BigEndian.Uint32(pkt[:HEAD_SIZE])

	if (s.maxRecvSize != 0 && size > s.maxRecvSize) || int(size) < MSG_ID_SIZE || int(size)+HEAD_SIZE != len(pkt) {
		log.Warnf("error parse packet,bytes:%d,size:%d,session:%d,remote:%s", len(pkt), size, s.SessionId, s.RemoteAddr())
		return nil
	}
	data := pkt[HEAD_SIZE:]
	// 解密
	if s.Flag&SESS_ENCRYPT != 0 {
		decoder, _ := rc4.NewCipher(s.DecodeKey)
		decoder.XORKeyStream(data, data)
	}
	msgId := binary.BigEndian.Uint16(data[:MSG_ID_SIZE])
	return &NetPacket{msgId, data[MSG_ID_SIZE:], s}
}

func (s *TCPSession) TraceStart(family, title string, expvar bool) {
	if trace.EnableTracing {
		s.TraceFinish(nil)
		s.tr = golangtrace.New(family, title, expvar)
	}
}

func (s *TCPSession) TraceFinish(traceDefer func(*expvar.Map, int64)) {
	if s.tr != nil {
		tt := s.tr
		tt.Finish()
		if traceDefer != nil {
			family := tt.GetFamily()
			req := expvar.Get(family)
			if req == nil {
				req = expvar.NewMap(family)
			}
			traceDefer(req.(*expvar.Map), tt.GetElapsedTime())
		}
		s.tr = nil
	}
}

func (s *TCPSession) TracePrintf(format string, a ...interface{}) {
	if s.tr != nil {
		s.tr.LazyPrintf(format, a...)
	}
}

func (s *TCPSession) TraceErrorf(format string, a ...interface{}) {
	if s.tr != nil {
		s.tr.LazyPrintf(format, a...)
		s.tr.SetError()
	}
}
