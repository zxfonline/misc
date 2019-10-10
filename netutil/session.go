package netutil

import (
	"crypto/rc4"
	"encoding/binary"
	"io"
	"net"
	"sync/atomic"
	"time"

	"topcrown.com/centerserver/misc/chanutil"

	"topcrown.com/centerserver/misc/log"
	"topcrown.com/centerserver/misc/timefix"
)

var _session_id int64

const (
	HEAD_SIZE   = 4
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
	Conn            NetConnIF //*net.TCPConn
	SendChan        chan *NetPacket
	ReadChan        chan *NetPacket
	CloseChan       chan bool // 关闭管道
	OffChan         chan int64
	SessionId       int64  // ID
	OffLine         bool   // 是否掉线
	Flag            int32  // 会话标记
	send_cache      []byte // 发送缓冲
	send_cache_size uint32 //发送缓冲大小

	read_delay    time.Duration
	send_dalay    time.Duration
	sendfullClose bool //发送管道满后是否需要关闭连接
	max_recv_size uint32
	rpm_limit     uint32     // 包频率包数
	rpm_interval  int64      // 包频率检测间隔
	off_line_msg  *NetPacket // 超过频率控制离线通知包

	OnLineTime  int64
	OffLineTime int64

	EncodeKey      []byte
	DecodeKey      []byte
	CloseStateChan chanutil.DoneChan
}

func (self *TCPSession) HandleConn() {
	go self.ReadLoop(nil)
	go self.Sendloop()
}
func (self *TCPSession) HandleConnFilter(filter func(*NetPacket) bool) {
	go self.ReadLoop(filter)
	go self.Sendloop()
}

//网络连接远程ip
func (self *TCPSession) NetIp() string {
	addr := self.Conn.RemoteAddr().String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	return net.ParseIP(host).String()
}

func (self *TCPSession) Send(packet *NetPacket) bool {
	if self.OffLine || packet == nil {
		return false
	}

	if wait := len(self.SendChan); wait > 400 && wait%50 == 0 {
		log.Warnf("session send process,ip:%s,msg:%d,waitchan:%d/%d", self.NetIp(), packet.MsgId, wait, cap(self.SendChan))
	}
	if !self.sendfullClose {
		//		t := time.NewTimer(3 * time.Second)
		select {
		case self.SendChan <- packet:
			//			t.Stop()
			return true
		//		case <-t.C:
		//			log.Warnf("send chan to session time out, msg:%d,waitchan:%d/%d", packet.MsgId, len(self.SendChan), cap(self.SendChan))
		//			return false
		default:
			log.Warnf("send chan to session time out, msg:%d,waitchan:%d/%d", packet.MsgId, len(self.SendChan), cap(self.SendChan))
			return false
		}
	} else {
		select {
		case self.SendChan <- packet:
			return true
		default:
			log.Errorf("session send channel full,ip:%s,send:%d/%d,read:%d/%d,msgid:%d", self.NetIp(), len(self.SendChan), cap(self.SendChan), len(self.ReadChan), cap(self.ReadChan), packet.MsgId)
			self.Close()
			return false
		}
	}
}

// RC4加密解密
func (self *TCPSession) SetCipher(encodeKey, decodeKey []byte) error {
	if len(encodeKey) < 1 || len(encodeKey) > 256 {
		return rc4.KeySizeError(len(encodeKey))
	}

	if len(decodeKey) < 1 || len(decodeKey) > 256 {
		return rc4.KeySizeError(len(decodeKey))
	}

	self.EncodeKey = encodeKey
	self.DecodeKey = decodeKey
	self.Flag |= SESS_KEYEXCG

	return nil
}
func CheckPanic() {
	if x := recover(); x != nil {
		log.Errorf("%v", x)
	}
}
func (self *TCPSession) ReadLoop(filter func(*NetPacket) bool) {
	defer CheckPanic()

	// 关闭发送
	defer self.Close()

	// 4字节包长度
	header := make([]byte, HEAD_SIZE)

	var delayTimer *time.Timer

	if self.read_delay > 0 {
		delayTimer = time.NewTimer(self.read_delay * time.Second)
	}

	rpmstart := time.Now().Unix()
	rpmCount := uint32(0)

	rpmMsgCount := 0

	for {
		// 读取超时
		if self.read_delay > 0 {
			self.Conn.SetReadDeadline(time.Now().Add(self.read_delay * time.Second))
		}

		// 4字节包长度
		n, err := io.ReadFull(self.Conn, header)
		if err != nil {
			//			if err != io.EOF {
			//				log.Warn("error receiving header, bytes:", n, " sessionId:", self.SessionId, "reason:", err)
			//			}
			return
		}

		// packet data
		size := binary.BigEndian.Uint32(header)
		if self.max_recv_size != 0 && size > self.max_recv_size {
			//			log.Warn("error receiving size:", size, " sessionId:", self.SessionId)
			return
		}

		data := make([]byte, size)
		n, err = io.ReadFull(self.Conn, data)
		if err != nil || size < MSG_ID_SIZE {
			log.Warn("error receiving msg, bytes:", n, "size:", size, " sessionId:", self.SessionId, "reason:", err)
			return
		}
		// 收包频率控制
		if self.rpm_limit > 0 {
			rpmCount++

			// 达到限制包数
			if rpmCount > self.rpm_limit {
				now := time.Now().Unix()
				// 检测时间间隔
				if now-rpmstart < self.rpm_interval {
					// 发送频率过高的消息包
					self.Send(self.off_line_msg)

					// 提示操作太频繁三次后踢下线
					rpmMsgCount++
					if rpmMsgCount > 3 {
						log.Error("session RPM too high ", rpmCount, " in ", self.rpm_interval, "s sessionId:", self.SessionId)
						return
					}
				}

				// 未超过限制
				rpmCount = 0
				rpmstart = now
			}
		}

		// 解密
		if self.Flag&SESS_ENCRYPT != 0 {
			decoder, _ := rc4.NewCipher(self.DecodeKey)
			decoder.XORKeyStream(data, data)
		}

		msgId := binary.BigEndian.Uint16(data[:MSG_ID_SIZE])

		pack := &NetPacket{msgId, data[MSG_ID_SIZE:], self}

		if self.read_delay > 0 {
			delayTimer.Reset(self.read_delay * time.Second)
			if filter != nil {
				if ok := filter(pack); !ok {
					select {
					case self.ReadChan <- pack:
					case <-delayTimer.C:
						log.Warn("session read busy or closed, ip:", self.NetIp(), " len:", len(self.ReadChan))
						return
					}
				}
			} else {
				select {
				case self.ReadChan <- pack:
				case <-delayTimer.C:
					log.Warn("session read busy or closed, ip:", self.NetIp(), " len:", len(self.ReadChan))
					return
				}
			}
		} else {
			if filter != nil {
				if ok := filter(pack); !ok {
					self.ReadChan <- pack
				}
			} else {
				self.ReadChan <- pack
			}
		}
	}
}

func (self *TCPSession) closetask() {
	self.OffLine = true
	self.OffLineTime = timefix.SecondTime()
	self.CloseStateChan.SetDone()

	// 告诉服务器该玩家掉线,或者下线
	t := time.NewTimer(15 * time.Second)
	select {
	case self.OffChan <- self.SessionId:
		t.Stop()
	case <-t.C:
		log.Warn("off chan time out, sessionId:", self.SessionId)
	}

	// close connection
	self.Conn.Close()
}
func (self *TCPSession) Sendloop() {
	defer CheckPanic()

	for {
		select {
		case <-self.CloseChan:
			self.closetask()
			return
		default:
			select {
			case packet := <-self.SendChan:
				self.DirectSend(packet)
			case <-self.CloseChan:
				self.closetask()
				return
			}
		}
	}
}
func (self *TCPSession) DirectSend(packet *NetPacket) {
	if packet == nil {
		return
	}
	packLen := uint32(len(packet.Data) + MSG_ID_SIZE)

	if packLen > self.send_cache_size {
		self.send_cache_size = packLen + (packLen >> 2) //1.25倍率
		self.send_cache = make([]byte, self.send_cache_size+HEAD_SIZE)
	}

	// 4字节包长度
	binary.BigEndian.PutUint32(self.send_cache, packLen)

	// 2字节消息id
	binary.BigEndian.PutUint16(self.send_cache[HEAD_SIZE:], packet.MsgId)

	copy(self.send_cache[HEAD_SIZE+MSG_ID_SIZE:], packet.Data)

	// encryption
	// (NOT_ENCRYPTED) -> KEYEXCG -> ENCRYPT
	if self.Flag&SESS_ENCRYPT != 0 { // encryption is enabled
		encoder, _ := rc4.NewCipher(self.EncodeKey)
		data := self.send_cache[HEAD_SIZE : HEAD_SIZE+packLen]

		encoder.XORKeyStream(data, data)
	} else if self.Flag&SESS_KEYEXCG != 0 { // key is exchanged, encryption is not yet enabled
		self.Flag &^= SESS_KEYEXCG
		self.Flag |= SESS_ENCRYPT
	}

	// 写超时
	if self.send_dalay > 0 {
		self.Conn.SetWriteDeadline(time.Now().Add(self.send_dalay * time.Second))
	}

	n, err := self.Conn.Write(self.send_cache[:HEAD_SIZE+packLen])
	if err != nil {
		log.Debug("error writing msg, bytes:", n, " sessionId:", self.SessionId, "reason:", err)
		self.Close()
	}
}
func (self *TCPSession) Close() {
	// 防止重复调用阻塞
	if !self.OffLine {
		self.OffLine = true
		if len(self.CloseChan) < cap(self.CloseChan) {
			self.CloseChan <- true
		}
	}
}

// 设置链接参数
func (self *TCPSession) SetParameter(read_delay, send_dalay time.Duration, maxRecvSize uint32, sendfullClose bool) {
	self.max_recv_size = maxRecvSize
	if read_delay >= 0 {
		self.read_delay = read_delay
	}
	if send_dalay >= 0 {
		self.send_dalay = send_dalay
	}
	self.sendfullClose = sendfullClose
}

// 包频率控制参数
func (self *TCPSession) SetRpmParameter(rpm_limit uint32, rpm_interval int64, msg *NetPacket) {
	self.rpm_limit = rpm_limit
	self.rpm_interval = rpm_interval
	self.off_line_msg = msg
}

func NewSession(conn NetConnIF, readChan, sendChan chan *NetPacket, offChan chan int64) *TCPSession {
	sess := TCPSession{}
	sess.Conn = conn
	sess.SendChan = sendChan
	sess.ReadChan = readChan
	sess.CloseChan = make(chan bool, 10)
	sess.OffChan = offChan
	sess.OffLine = false
	sess.SessionId = atomic.AddInt64(&_session_id, -1)
	sess.send_cache_size = PACKET_LIMIT
	sess.send_cache = make([]byte, sess.send_cache_size+HEAD_SIZE)

	// 默认设置
	sess.max_recv_size = 0
	sess.rpm_limit = 0 // 默认无限制
	sess.rpm_interval = 0
	sess.read_delay = 120
	sess.send_dalay = 60
	sess.OnLineTime = timefix.SecondTime()
	sess.CloseStateChan = chanutil.NewDoneChan()

	return &sess
}

func (self *TCPSession) ParsePacket(pkt []byte) *NetPacket {
	defer CheckPanic()
	if len(pkt) <= HEAD_SIZE {
		log.Warn("error parse packet bytes:", len(pkt), " sessionId:", self.SessionId)
		return nil
	}
	// 4字节包长度
	// packet data
	size := binary.BigEndian.Uint32(pkt[:HEAD_SIZE])

	if (self.max_recv_size != 0 && size > self.max_recv_size) || int(size) < MSG_ID_SIZE || int(size)+HEAD_SIZE != len(pkt) {
		log.Warn("error parse packet msg, bytes:", len(pkt), "size:", size, " sessionId:", self.SessionId)
		return nil
	}
	data := pkt[HEAD_SIZE:]
	// 解密
	if self.Flag&SESS_ENCRYPT != 0 {
		decoder, _ := rc4.NewCipher(self.DecodeKey)
		decoder.XORKeyStream(data, data)
	}
	msgId := binary.BigEndian.Uint16(data[:MSG_ID_SIZE])
	return &NetPacket{msgId, data[MSG_ID_SIZE:], self}
}
