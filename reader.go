package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

const (
	BYTES_IN_INT   = 4
	BYTES_IN_SHORT = 2
	BITS_IN_SHORT  = 16
	BITS_IN_BYTE   = 8
	BYTE_MASK      = 0x07
	BIT_MASK       = 0x01

	BITS_IN_MINIFLOAT    = 13
	BITS_IN_ENTITY_INDEX = 10
	MAX_ENTITY_INDEX     = (1 << BITS_IN_ENTITY_INDEX) - 1
)

/* -------------------------------------------- */
type BitReader struct {
	reader   *bytes.Reader
	buffered byte
	offset   int
}

func NewBitReader(reader *bytes.Reader) *BitReader {
	return &BitReader{reader: reader}
}

func (self *BitReader) Position() int {
	return self.offset
}

func (self *BitReader) Remaining() int {
	tmp1 := self.reader.Len() * BITS_IN_BYTE
	tmp2 := int((BITS_IN_BYTE - (self.offset & BYTE_MASK)) & BYTE_MASK)
	return tmp1 + tmp2
}

func (self *BitReader) Read() int {
	bit_offset := uint(self.offset & BYTE_MASK)
	if bit_offset == 0 {
		self.buffered, _ = self.reader.ReadByte()
	}
	self.offset += 1
	return int((self.buffered >> bit_offset) & BIT_MASK)
}

/* -------------------------------------------- */
type BitDecoder struct {
	reader *BitReader
}

func NewBitDecoder(reader *BitReader) *BitDecoder {
	return &BitDecoder{reader: reader}
}

func (self *BitDecoder) Read() int {
	return self.decode(1)
}

func (self *BitDecoder) decode(index int) int {
	if index < len(decoderTree) && decoderTree[index] >= 0 {
		return int(decoderTree[index])
	} else if index < len(decoderTree) {
		return self.decode(index*2 + self.reader.Read())
	} else {
		return 0
	}
}

/* -------------------------------------------- */
type DataReader struct {
	reader  *BitReader
	decoder *BitDecoder
}

func NewDataReader(reader *BitReader) *DataReader {
	return &DataReader{reader: reader, decoder: NewBitDecoder(reader)}
}

func (self *DataReader) ReadBit() int {
	return self.reader.Read()
}

func (self *DataReader) ReadBitsSub8(count int) int {
	if count == 0 {
		return 0
	} else {
		return self.ReadBit() | (self.ReadBitsSub8(count-1) << 1)
	}
}

func (self *DataReader) ReadBits(count int) int {
	bits := count & BYTE_MASK
	bytes := count / BITS_IN_BYTE
	return self.ReadBitsSub8(bits) | (self.ReadBytes(bytes) << uint(bits))
}

func (self *DataReader) ReadByte() int {
	return self.decoder.Read()
}

func (self *DataReader) ReadBytes(count int) int {
	if count == 0 {
		return 0
	} else {
		return self.ReadByte() | (self.ReadBytes(count-1) << BITS_IN_BYTE)
	}
}

func (self *DataReader) ReadShort() int {
	return self.ReadBytes(BYTES_IN_SHORT)
}

func (self *DataReader) ReadInt() int {
	return self.ReadBytes(BYTES_IN_INT)
}

func (self *DataReader) ReadSignedByte() int {
	return int(int8(self.ReadByte()))
}

func (self *DataReader) ReadSignedShort() int {
	return int(int16(self.ReadShort()))
}

func (self *DataReader) ReadFloat() float32 {
	if self.ReadBit() == 0 {
		return float32(self.ReadBits(BITS_IN_MINIFLOAT) - (1 << (BITS_IN_MINIFLOAT - 1)))
	} else {
		tmp := uint32(self.ReadInt())
		buffer := new(bytes.Buffer)
		binary.Write(buffer, binary.LittleEndian, tmp)
		var ret float32
		binary.Read(buffer, binary.LittleEndian, &ret)
		return ret
	}
}

func (self *DataReader) ReadString() string {
	buffer := new(bytes.Buffer)
	for {
		tmp := self.ReadByte()
		if tmp == 0 {
			return string(buffer.Bytes())
		}
		buffer.WriteByte(byte(tmp))
	}
}

func (self *DataReader) ReadBlob(count int) []byte {
	buffer := new(bytes.Buffer)
	for i := 0; i < count; i += 1 {
		buffer.WriteByte(byte(self.ReadByte()))
	}
	return buffer.Bytes()
}

/* -------------------------------------------- */
type StateReader struct {
	reader *DataReader
}

func NewStateReader(reader *DataReader) *StateReader {
	return &StateReader{reader: reader}
}

func (self *StateReader) ReadTemplate(read func() int) int {
	if self.reader.ReadBit() == 0 {
		return 0
	} else {
		return read()
	}
}

func (self *StateReader) ReadBits(count int) int {
	return self.ReadTemplate(func() int { return self.reader.ReadBits(count) })
}

func (self *StateReader) ReadByte() int {
	return self.ReadTemplate(self.reader.ReadByte)
}

func (self *StateReader) ReadShort() int {
	return self.ReadTemplate(self.reader.ReadShort)
}

func (self *StateReader) ReadInt() int {
	return self.ReadTemplate(self.reader.ReadInt)
}

func (self *StateReader) ReadFloat() float32 {
	if self.reader.ReadBit() == 0 {
		return 0
	} else {
		return self.reader.ReadFloat()
	}
}

func (self *StateReader) ReadValues(readers []func()) {
	if self.reader.ReadBit() == 0 {
		return
	}
	flags := self.reader.ReadShort()
	for i := uint(0); i < 16; i += 1 {
		if flags&(1<<i) != 0 {
			readers[i]()
		}
	}
}

func (self *StateReader) ReadEntity(result *Entity) {
	if self.reader.ReadBit() == 0 {
		return
	}

	readers := []func(){
		func() { result.Trajectories.A.Time = self.ReadInt() },
		func() { result.Trajectories.A.Base.X = self.ReadFloat() },
		func() { result.Trajectories.A.Base.Y = self.ReadFloat() },
		func() { result.Trajectories.A.Delta.X = self.ReadFloat() },
		func() { result.Trajectories.A.Delta.Y = self.ReadFloat() },
		func() { result.Trajectories.A.Base.Z = self.ReadFloat() },
		func() { result.Trajectories.B.Base.Y = self.ReadFloat() },
		func() { result.Trajectories.A.Delta.Z = self.ReadFloat() },
		func() { result.Trajectories.B.Base.X = self.ReadFloat() },
		func() { result.Trajectories.A.Gravity = self.ReadInt() },
		func() { result.Events.A = self.ReadBits(BITS_IN_ENTITY_INDEX) },
		func() { result.Angles.B.Y = self.ReadFloat() },
		func() { result.Entity.B = self.ReadByte() },
		func() { result.Animations.A = self.ReadByte() },
		func() { result.Events.B = self.ReadByte() },
		func() { result.Animations.B = self.ReadByte() },
		func() { result.Entities.A = self.ReadBits(BITS_IN_ENTITY_INDEX) },
		func() { result.Trajectories.A.Mode = self.ReadByte() },
		func() { result.Entity.A = self.ReadBits(19) },
		func() { result.Entities.B = self.ReadBits(BITS_IN_ENTITY_INDEX) },
		func() { result.Weapon = self.ReadByte() },
		func() { result.Client = self.ReadByte() },
		func() { result.Angles.A.Y = self.ReadFloat() },
		func() { result.Trajectories.A.Duration = self.ReadInt() },
		func() { result.Trajectories.B.Mode = self.ReadByte() },
		func() { result.Origins.A.X = self.ReadFloat() },
		func() { result.Origins.A.Y = self.ReadFloat() },
		func() { result.Origins.A.Z = self.ReadFloat() },
		func() { result.Misc.E = self.ReadBits(24) },
		func() { result.Powerups = self.ReadShort() },
		func() { result.Model.A = self.ReadByte() },
		func() { result.Entities.C = self.ReadBits(BITS_IN_ENTITY_INDEX) },
		func() { result.Misc.D = self.ReadByte() },
		func() { result.Misc.C = self.ReadByte() },
		func() { result.Origins.B.Z = self.ReadFloat() },
		func() { result.Origins.B.X = self.ReadFloat() },
		func() { result.Origins.B.Y = self.ReadFloat() },
		func() { result.Model.B = self.ReadByte() },
		func() { result.Angles.A.X = self.ReadFloat() },
		func() { result.Time.A = self.ReadInt() },
		func() { result.Trajectories.B.Time = self.ReadInt() },
		func() { result.Trajectories.B.Duration = self.ReadInt() },
		func() { result.Trajectories.B.Base.Z = self.ReadFloat() },
		func() { result.Trajectories.B.Delta.X = self.ReadFloat() },
		func() { result.Trajectories.B.Delta.Y = self.ReadFloat() },
		func() { result.Trajectories.B.Delta.Z = self.ReadFloat() },
		func() { result.Trajectories.B.Gravity = self.ReadInt() },
		func() { result.Time.B = self.ReadInt() },
		func() { result.Angles.A.Z = self.ReadFloat() },
		func() { result.Angles.B.X = self.ReadFloat() },
		func() { result.Angles.B.Z = self.ReadFloat() },
		func() { result.Misc.A = self.ReadInt() },
		func() { result.Misc.B = self.ReadShort() },
	}

	count := self.reader.ReadByte()
	for i := 0; i < count; i += 1 {
		if self.reader.ReadBit() == 1 {
			readers[i]()
		}
	}
}

func (self *StateReader) ReadPlayer(result *Player) {
	readers := []func(){
		func() { result.Time = self.reader.ReadInt() },
		func() { result.Origin.X = self.reader.ReadFloat() },
		func() { result.Origin.Y = self.reader.ReadFloat() },
		func() { result.Misc.A = self.reader.ReadByte() },
		func() { result.Velocity.X = self.reader.ReadFloat() },
		func() { result.Velocity.Y = self.reader.ReadFloat() },
		func() { result.View.Y = self.reader.ReadFloat() },
		func() { result.View.X = self.reader.ReadFloat() },
		func() { result.Weapon.C = self.reader.ReadSignedShort() },
		func() { result.Origin.Z = self.reader.ReadFloat() },
		func() { result.Velocity.Z = self.reader.ReadFloat() },
		func() { result.Animations.B.B = self.reader.ReadByte() },
		func() { result.Movement.C = self.reader.ReadSignedShort() },
		func() { result.Event.A = self.reader.ReadShort() },
		func() { result.Animations.A.A = self.reader.ReadByte() },
		func() { result.Movement.A = self.reader.ReadBits(4) },
		func() { result.Events.A = self.reader.ReadByte() },
		func() { result.Animations.B.A = self.reader.ReadByte() },
		func() { result.Events.B = self.reader.ReadByte() },
		func() { result.Movement.B = self.reader.ReadShort() },
		func() { result.Entities.A = self.reader.ReadBits(BITS_IN_ENTITY_INDEX) },
		func() { result.Weapon.B = self.reader.ReadBits(4) },
		func() { result.Entity.A = self.reader.ReadShort() },
		func() { result.External.A = self.reader.ReadBits(BITS_IN_ENTITY_INDEX) },
		func() { result.Misc.C = self.reader.ReadShort() },
		func() { result.Misc.E = self.reader.ReadShort() },
		func() { result.Delta.B = self.reader.ReadShort() },
		func() { result.External.B = self.reader.ReadByte() },
		func() { result.Misc.F = self.reader.ReadSignedByte() },
		func() { result.Damage.B = self.reader.ReadByte() },
		func() { result.Damage.D = self.reader.ReadByte() },
		func() { result.Damage.C = self.reader.ReadByte() },
		func() { result.Damage.A = self.reader.ReadByte() },
		func() { result.Misc.C = self.reader.ReadByte() },
		func() { result.Movement.D = self.reader.ReadByte() },
		func() { result.Delta.A = self.reader.ReadShort() },
		func() { result.Delta.C = self.reader.ReadShort() },
		func() { result.Animations.A.B = self.reader.ReadBits(12) },
		func() { result.Event.B = self.reader.ReadByte() },
		func() { result.Event.C = self.reader.ReadByte() },
		func() { result.Client = self.reader.ReadByte() },
		func() { result.Weapon.A = self.reader.ReadBits(5) },
		func() { result.View.Z = self.reader.ReadFloat() },
		func() { result.Grapple.X = self.reader.ReadFloat() },
		func() { result.Grapple.Y = self.reader.ReadFloat() },
		func() { result.Grapple.Z = self.reader.ReadFloat() },
		func() { result.Entities.B = self.reader.ReadBits(BITS_IN_ENTITY_INDEX) },
		func() { result.Misc.D = self.reader.ReadShort() },
	}

	count := self.reader.ReadByte()
	for i := 0; i < count; i += 1 {
		if self.reader.ReadBit() == 1 {
			readers[i]()
		}
	}

	if self.reader.ReadBit() == 1 {
		{
			readers := []func(){
				func() { result.Vitals.A = self.reader.ReadSignedShort() },
				func() { result.Vitals.B = self.reader.ReadSignedShort() },
				func() { result.Vitals.C = self.reader.ReadSignedShort() },
				func() { result.Vitals.D = self.reader.ReadSignedShort() },
				func() { result.Vitals.E = self.reader.ReadSignedShort() },
				func() { result.Vitals.F = self.reader.ReadSignedShort() },
				func() { result.Vitals.G = self.reader.ReadSignedShort() },
				func() { result.Vitals.H = self.reader.ReadSignedShort() },
				func() { result.Vitals.I = self.reader.ReadSignedShort() },
				func() { result.Vitals.J = self.reader.ReadSignedShort() },
				func() { result.Vitals.K = self.reader.ReadSignedShort() },
				func() { result.Vitals.L = self.reader.ReadSignedShort() },
				func() { result.Vitals.M = self.reader.ReadSignedShort() },
				func() { result.Vitals.N = self.reader.ReadSignedShort() },
				func() { result.Vitals.O = self.reader.ReadSignedShort() },
				func() { result.Vitals.P = self.reader.ReadSignedShort() },
			}
			self.ReadValues(readers)
		}

		{
			readers := []func(){
				func() { result.Attributes.A = self.reader.ReadShort() },
				func() { result.Attributes.B = self.reader.ReadShort() },
				func() { result.Attributes.C = self.reader.ReadShort() },
				func() { result.Attributes.D = self.reader.ReadShort() },
				func() { result.Attributes.E = self.reader.ReadShort() },
				func() { result.Attributes.F = self.reader.ReadShort() },
				func() { result.Attributes.G = self.reader.ReadShort() },
				func() { result.Attributes.H = self.reader.ReadShort() },
				func() { result.Attributes.I = self.reader.ReadShort() },
				func() { result.Attributes.J = self.reader.ReadShort() },
				func() { result.Attributes.K = self.reader.ReadShort() },
				func() { result.Attributes.L = self.reader.ReadShort() },
				func() { result.Attributes.M = self.reader.ReadShort() },
				func() { result.Attributes.N = self.reader.ReadShort() },
				func() { result.Attributes.O = self.reader.ReadShort() },
				func() { result.Attributes.P = self.reader.ReadShort() },
			}
			self.ReadValues(readers)
		}

		{
			readers := []func(){
				func() { result.Ammunition.A = self.reader.ReadSignedShort() },
				func() { result.Ammunition.B = self.reader.ReadSignedShort() },
				func() { result.Ammunition.C = self.reader.ReadSignedShort() },
				func() { result.Ammunition.D = self.reader.ReadSignedShort() },
				func() { result.Ammunition.E = self.reader.ReadSignedShort() },
				func() { result.Ammunition.F = self.reader.ReadSignedShort() },
				func() { result.Ammunition.G = self.reader.ReadSignedShort() },
				func() { result.Ammunition.H = self.reader.ReadSignedShort() },
				func() { result.Ammunition.I = self.reader.ReadSignedShort() },
				func() { result.Ammunition.J = self.reader.ReadSignedShort() },
				func() { result.Ammunition.K = self.reader.ReadSignedShort() },
				func() { result.Ammunition.L = self.reader.ReadSignedShort() },
				func() { result.Ammunition.M = self.reader.ReadSignedShort() },
				func() { result.Ammunition.N = self.reader.ReadSignedShort() },
				func() { result.Ammunition.O = self.reader.ReadSignedShort() },
				func() { result.Ammunition.P = self.reader.ReadSignedShort() },
			}
			self.ReadValues(readers)
		}

		{
			readers := []func(){
				func() { result.Powerups.A = self.reader.ReadInt() },
				func() { result.Powerups.B = self.reader.ReadInt() },
				func() { result.Powerups.C = self.reader.ReadInt() },
				func() { result.Powerups.D = self.reader.ReadInt() },
				func() { result.Powerups.E = self.reader.ReadInt() },
				func() { result.Powerups.F = self.reader.ReadInt() },
				func() { result.Powerups.G = self.reader.ReadInt() },
				func() { result.Powerups.H = self.reader.ReadInt() },
				func() { result.Powerups.I = self.reader.ReadInt() },
				func() { result.Powerups.J = self.reader.ReadInt() },
				func() { result.Powerups.K = self.reader.ReadInt() },
				func() { result.Powerups.L = self.reader.ReadInt() },
				func() { result.Powerups.M = self.reader.ReadInt() },
				func() { result.Powerups.N = self.reader.ReadInt() },
				func() { result.Powerups.O = self.reader.ReadInt() },
				func() { result.Powerups.P = self.reader.ReadInt() },
			}
			self.ReadValues(readers)
		}
	}
}

/* -------------------------------------------- */
type DemoState struct {
	Player          Player
	Entities        map[int]*Entity
	EntityBaselines map[int]*Entity
	Config          map[int]string
	configTmp       map[int]string
}

func NewDemoState() *DemoState {
	return &DemoState{Entities: make(map[int]*Entity), EntityBaselines: make(map[int]*Entity),
		Config: make(map[int]string), configTmp: make(map[int]string)}
}

func (self *DemoState) OnBaselineConfig(id int, str string) {
	self.Config[id] = str
}

func (self *DemoState) OnBaselineEntity(id int, entity *Entity) {
	self.Entities[id] = entity
	self.EntityBaselines[id] = entity
}

func (self *DemoState) OnEntityRemoved(id int) {
	delete(self.Entities, id)
}

func (self *DemoState) OnEntityUpdate(id int) {
	if _, ok := self.Entities[id]; !ok {
		self.Entities[id] = new(Entity)
	}
	if _, ok := self.EntityBaselines[id]; ok {
		*self.Entities[id] = *self.EntityBaselines[id]
	}
}

var csRegexp = regexp.MustCompile(`(?ms)^cs (\d+) "(.*?)".*?$`)
var bcsRegexp = regexp.MustCompile(`(?ms)^bcs(\d) (\d+) "(.+?)".*?$`)

func (self *DemoState) OnMessageCommand(id int, str string) {
	if matches := csRegexp.FindStringSubmatch(str); matches != nil {
		code, _ := strconv.Atoi(matches[0])
		self.Config[code] = matches[1]
	} else if matches := bcsRegexp.FindStringSubmatch(str); matches != nil {
		index, _ := strconv.Atoi(matches[0])
		code, _ := strconv.Atoi(matches[1])
		if index == 0 {
			self.configTmp[code] = matches[2]
		} else {
			self.configTmp[code] = strings.Join([]string{self.configTmp[code], matches[2]}, "")
		}
		if index == 3 {
			self.Config[code] = self.configTmp[code]
			delete(self.configTmp, code)
		} else if index > 3 {
			panic("index cannot be larger than 3")
		}
	}
}

/* -------------------------------------------- */
type DemoReader struct {
	reader      io.Reader
	demoState   *DemoState
	dataReader  *DataReader
	stateReader *StateReader
}

func NewDemoReader(reader io.Reader, demoState *DemoState) *DemoReader {
	return &DemoReader{reader: reader, demoState: demoState}
}

func (self *DemoReader) ReadBytes(count int) []byte {
	buffer := make([]byte, count)
	for count > 0 {
		read, err := self.reader.Read(buffer[len(buffer)-count:])
		if err != nil {
			panic("Reading failed")
		}
		if read == 0 && len(buffer) == count {
			return nil
		}
		count -= read
	}
	return buffer
}

func (self *DemoReader) ReadInt() (bool, int) {
	tmp := self.ReadBytes(BYTES_IN_INT)
	if tmp == nil {
		return false, 0
	} else {
		var ret int32
		binary.Read(bytes.NewReader(tmp), binary.LittleEndian, &ret)
		return true, int(ret)
	}
}

func (self *DemoReader) ReadBlock() []byte {
	ok, length := self.ReadInt()
	if !ok || length == -1 {
		return nil
	} else {
		return self.ReadBytes(length)
	}
}

func (self *DemoReader) BlockLoop(channel chan interface{}) {
	for {
		ok, blockId := self.ReadInt()
		if !ok {
			close(channel)
			return
		}

		block := self.ReadBlock()
		if block == nil {
			close(channel)
			return
		}

		bitReader := NewBitReader(bytes.NewReader(block))
		self.dataReader = NewDataReader(bitReader)
		self.stateReader = NewStateReader(self.dataReader)

		messageId := self.dataReader.ReadInt()
		self.MessageLoop(channel, blockId, messageId)
	}
}

func (self *DemoReader) MessageLoop(channel chan interface{}, blockId, messageId int) {
	for {
		switch tmp := self.dataReader.ReadByte(); tmp {
		case 1:
		case 2:
			id := self.dataReader.ReadInt()
			client, checksum := self.GamestateLoop()
			channel <- &Gamestate{id, client, checksum}
		case 5:
			id := self.dataReader.ReadInt()
			str := self.dataReader.ReadString()
			self.demoState.OnMessageCommand(id, str)
			channel <- &Command{id, str}
		case 7:
			time := self.dataReader.ReadInt()
			delta := self.dataReader.ReadByte()
			flags := self.dataReader.ReadByte()
			blob_len := self.dataReader.ReadByte()
			blob := self.dataReader.ReadBlob(blob_len)
			self.stateReader.ReadPlayer(&self.demoState.Player)
			self.SnapshotLoop()
			channel <- &Snapshot{time, delta, flags, blob}
		case 8:
			return
		default:
			panic(fmt.Sprintf("Invalid message loop code: %v", tmp))
		}
	}
}

func (self *DemoReader) GamestateLoop() (int, int) {
	for {
		switch tmp := self.dataReader.ReadByte(); tmp {
		case 3:
			id := self.dataReader.ReadShort()
			str := self.dataReader.ReadString()
			self.demoState.OnBaselineConfig(id, str)
		case 4:
			id := self.dataReader.ReadBits(BITS_IN_ENTITY_INDEX)
			entity := new(Entity)
			if self.dataReader.ReadBit() == 0 {
				self.stateReader.ReadEntity(entity)
			}
			self.demoState.OnBaselineEntity(id, entity)
		case 8:
			client := self.dataReader.ReadInt()
			checksum := self.dataReader.ReadInt()
			return client, checksum
		default:
			panic(fmt.Sprintf("Invalid gamestate loop code: %v", tmp))
		}
	}
}

func (self *DemoReader) SnapshotLoop() {
	for {
		id := self.dataReader.ReadBits(BITS_IN_ENTITY_INDEX)
		if id == MAX_ENTITY_INDEX {
			return
		}
		if self.dataReader.ReadBit() == 1 {
			self.demoState.OnEntityRemoved(id)
		} else {
			self.demoState.OnEntityUpdate(id)
			self.stateReader.ReadEntity(self.demoState.Entities[id])
		}
	}
}

func (self *DemoReader) Iterate() chan interface{} {
	result := make(chan interface{})
	go self.BlockLoop(result)
	return result
}
