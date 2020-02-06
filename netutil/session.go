package netutil

import (
	"crypto/rc4"
	"encoding/binary"
	"io"
	"net"
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
	//消息报头字节数
	HEAD_SIZE = 4
	//消息包自增的序号
	SEQ_ID_SIZE = 4
	//消息号占用的字节数
	MSG_ID_SIZE = 2
)

var (
	ServerEndian = binary.LittleEndian
)

const (
	SESS_KEYEXCG = 0x1 // 是否已经交换完毕KEY
	SESS_ENCRYPT = 0x2 // 是否可以开始加密
)

type NetPacket struct {
	MsgId   uint16
	Data    []byte
	Session *TCPSession
	//收到该消息包的时间戳 毫秒
	ReceiveTime time.Time
}

type NetConnIF interface {
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	Close() error
	SetWriteBuffer(bytes int) error
	SetReadBuffer(bytes int) error
	Write(b []byte) (n int, err error)
	RemoteAddr() net.Addr
	Read(p []byte) (n int, err error)
}

type TCPSession struct {
	//Conn *net.TCPConn
	Conn     NetConnIF
	IP       net.IP
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
	rpmInterval time.Duration
	// 对收到的包进行计数，可避免重放攻击-REPLAY-ATTACK
	PacketRcvSeq uint32
	//数据包发送计数器
	PacketSndSeq uint32
	// 超过频率控制离线通知包
	offLineMsg *NetPacket

	OnLineTime  int64
	OffLineTime int64
	EncodeKey   []byte
	DecodeKey   []byte

	tr       golangtrace.Trace
	sendLock sync.Mutex
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
func (s *TCPSession) RemoteIp() net.IP {
	//addr := s.Conn.RemoteAddr().String()
	//host, _, err := net.SplitHostPort(addr)
	//if err != nil {
	//	host = addr
	//}
	//return net.ParseIP(host).String()
	return s.IP
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

func (s *TCPSession) ReadLoop(filter func(*NetPacket) bool) {
	defer log.PrintPanicStack()

	// 关闭发送
	defer s.Close()

	//var delayTimer *time.Timer

	//if s.readDelay > 0 {
	//	delayTimer = time.NewTimer(s.readDelay)
	//}

	rpmStart := time.Now()
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

		// packet payload
		size := ServerEndian.Uint32(header)
		if size < SEQ_ID_SIZE+MSG_ID_SIZE || (s.maxRecvSize != 0 && size > s.maxRecvSize) {
			log.Warnf("error receiving,size:%d,head:%+v,session:%d,remote:%s", size, header, s.SessionId, s.RemoteAddr())
			return
		}
		payload := make([]byte, size)
		n, err = io.ReadFull(s.Conn, payload)
		if err != nil {
			log.Warnf("error receiving body,bytes:%d,size:%d,session:%d,remote:%s,err:%v", n, size, s.SessionId, s.RemoteAddr(), err)
			return
		}
		// 收包频率控制
		if s.rpmLimit > 0 {
			rpmCount++

			// 达到限制包数
			if rpmCount > s.rpmLimit {
				now := time.Now()
				// 检测时间间隔
				if now.Sub(rpmStart) < s.rpmInterval {
					// 提示操作太频繁三次后踢下线
					//rpmMsgCount++
					//if rpmMsgCount > 3 {
					// 发送频率过高的消息包
					s.DirectSendAndClose(s.offLineMsg)
					log.Errorf("session rpm too high,%d/%s qps,session:%d,remote:%s", rpmCount, s.rpmInterval, s.SessionId, s.RemoteAddr())
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
		s.PacketRcvSeq++

		//fmt.Printf("a   %x\n", payload)
		// 解密
		if s.Flag&SESS_ENCRYPT != 0 {
			decoder, _ := rc4.NewCipher(s.DecodeKey)
			decoder.XORKeyStream(payload, payload)
			//fmt.Printf("b1  %x\n", payload)
			//} else {
			//fmt.Printf("b2  %x\n", payload)
		}
		// 读客户端数据包序列号(1,2,3...)
		// 客户端发送的数据包必须包含一个自增的序号，必须严格递增
		// 加密后，可避免重放攻击-REPLAY-ATTACK
		seqId := ServerEndian.Uint32(payload[:SEQ_ID_SIZE])
		if seqId != s.PacketRcvSeq {
			log.Errorf("session illegal packet sequence id:%v should be:%v size:%v", seqId, s.PacketRcvSeq, len(payload))
			return
		}
		msgId := ServerEndian.Uint16(payload[SEQ_ID_SIZE : SEQ_ID_SIZE+MSG_ID_SIZE])

		pack := &NetPacket{MsgId: msgId, Data: payload[SEQ_ID_SIZE+MSG_ID_SIZE:], Session: s, ReceiveTime: time.Now()}

		//if s.readDelay > 0 {
		//	if !delayTimer.Stop() {
		//		select {
		//		case <-delayTimer.C:
		//		default:
		//		}
		//	}
		//	delayTimer.Reset(s.readDelay)
		//	if filter == nil {
		//		select {
		//		case s.ReadChan <- pack:
		//		case <-delayTimer.C:
		//			log.Warnf("session read busy or closed,waitChan:%d,session:%d,remote:%s", len(s.ReadChan), s.SessionId, s.RemoteAddr())
		//			return
		//		}
		//	} else {
		//		if ok := filter(pack); !ok {
		//			select {
		//			case s.ReadChan <- pack:
		//			case <-delayTimer.C:
		//				log.Warnf("session read busy or closed,waitChan:%d,session:%d,remote:%s", len(s.ReadChan), s.SessionId, s.RemoteAddr())
		//				return
		//			}
		//		}
		//	}
		//} else {
		if filter == nil {
			s.ReadChan <- pack
		} else {
			if ok := filter(pack); !ok {
				s.ReadChan <- pack
			}
		}
		//}
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
	if s.OffChan != nil {
		s.OffChan <- s.SessionId
		//// 告诉服务器该玩家掉线,或者下线
		//t := time.NewTimer(15 * time.Second)
		//select {
		//case s.OffChan <- s.SessionId:
		//	t.Stop()
		//case <-t.C:
		//	log.Warnf("off chan time out,session:%d,remote:%s", s.SessionId, s.RemoteAddr())
		//}
	}

	// close connection
	s.Conn.Close()
}
func (s *TCPSession) SendLoop() {
	defer log.PrintPanicStack()

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

func (s *TCPSession) DirectSendAndClose(packet *NetPacket) {
	go func() {
		defer s.Close()
		s.DirectSend(packet)
		time.Sleep(1 * time.Second)
	}()
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
	s.PacketSndSeq++
	packLen := uint32(SEQ_ID_SIZE + MSG_ID_SIZE + len(packet.Data))
	totalSize := packLen + HEAD_SIZE
	if totalSize > s.sendCacheSize {
		s.sendCacheSize = totalSize + (totalSize >> 2) //1.25倍率
		s.sendCache = make([]byte, s.sendCacheSize)
	}

	// 4字节包长度
	ServerEndian.PutUint32(s.sendCache, packLen)
	//4字节消息序列号
	ServerEndian.PutUint32(s.sendCache[HEAD_SIZE:], s.PacketSndSeq)
	// 2字节消息id
	ServerEndian.PutUint16(s.sendCache[HEAD_SIZE+SEQ_ID_SIZE:], packet.MsgId)

	copy(s.sendCache[HEAD_SIZE+SEQ_ID_SIZE+MSG_ID_SIZE:], packet.Data)

	// encryption
	// (NOT_ENCRYPTED) -> KEYEXCG -> ENCRYPT
	if s.Flag&SESS_ENCRYPT != 0 { // encryption is enabled
		encoder, _ := rc4.NewCipher(s.EncodeKey)
		data := s.sendCache[HEAD_SIZE:totalSize]
		encoder.XORKeyStream(data, data)
	} else if s.Flag&SESS_KEYEXCG != 0 { // key is exchanged, encryption is not yet enabled
		//s.Flag &^= SESS_KEYEXCG
		s.Flag |= SESS_ENCRYPT
	}

	err := s.performSend(s.sendCache[:totalSize], 0)
	if err != nil {
		log.Debugf("error writing msg,session:%d,remote:%s,err:%v", s.SessionId, s.RemoteAddr(), err)
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
func (s *TCPSession) SetRpmParameter(rpmLimit uint32, rpmInterval time.Duration, msg *NetPacket) {
	s.rpmLimit = rpmLimit
	s.rpmInterval = rpmInterval
	s.offLineMsg = msg
}

func NewSession(conn NetConnIF, readChan, sendChan chan *NetPacket, offChan chan int64) (*TCPSession, error) {
	s := &TCPSession{
		Conn:          conn,
		SendChan:      sendChan,
		ReadChan:      readChan,
		OffChan:       offChan,
		SessionId:     atomic.AddInt64(&_sessionID, 1),
		sendCache:     make([]byte, 256),
		sendCacheSize: 256,
		readDelay:     30 * time.Second,
		sendDelay:     10 * time.Second,
		sendFullClose: true,
		maxRecvSize:   10 * 1024,
		OnLineTime:    timefix.SecondTime(),
		CloseState:    chanutil.NewDoneChan(),
	}
	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		log.Error("cannot get remote address:", err)
		return nil, err
	}
	s.IP = net.ParseIP(host)
	//log.Debugf("new connection from:%v port:%v", host, port)
	return s, nil
}

func (s *TCPSession) TraceStart(family, title string, expvar bool) {
	if trace.EnableTracing {
		s.TraceFinish(nil)
		s.tr = golangtrace.New(family, title, expvar)
	}
}
func (s *TCPSession) TraceStartWithStart(family, title string, expvar bool, startTime time.Time) {
	if trace.EnableTracing {
		s.TraceFinish(nil)
		s.tr = golangtrace.NewWithStart(family, title, expvar, startTime)
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
				func() {
					defer func() {
						if v := recover(); v != nil {
							req = expvar.Get(family)
						}
					}()
					req = expvar.NewMap(family)
				}()
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
