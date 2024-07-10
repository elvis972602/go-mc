package world

import (
	"errors"
	"sync"

	"github.com/Tnze/go-mc/bot"
	"github.com/Tnze/go-mc/bot/basic"
	"github.com/Tnze/go-mc/data/packetid"
	"github.com/Tnze/go-mc/level"
	pk "github.com/Tnze/go-mc/net/packet"
)

type World struct {
	c      *bot.Client
	p      *basic.Player
	events EventsListener

	Columns map[level.ChunkPos]*level.Chunk

	ChunkLock sync.RWMutex
}

func NewWorld(c *bot.Client, p *basic.Player, events EventsListener) (w *World) {
	w = &World{
		c: c, p: p,
		events:  events,
		Columns: make(map[level.ChunkPos]*level.Chunk),
	}
	c.Events.AddListener(
		bot.PacketHandler{Priority: 64, ID: packetid.ClientboundLogin, F: w.onPlayerSpawn},
		bot.PacketHandler{Priority: 64, ID: packetid.ClientboundRespawn, F: w.onPlayerSpawn},
		bot.PacketHandler{Priority: 0, ID: packetid.ClientboundLevelChunkWithLight, F: w.handleLevelChunkWithLightPacket},
		bot.PacketHandler{Priority: 0, ID: packetid.ClientboundForgetLevelChunk, F: w.handleForgetLevelChunkPacket},
		bot.PacketHandler{Priority: 0, ID: packetid.ClientboundBlockUpdate, F: w.handleBlockUpdatePacket},
		bot.PacketHandler{Priority: 0, ID: packetid.ClientboundSectionBlocksUpdate, F: w.handleMultiBlockUpdatePacket},
	)
	return
}

func (w *World) IsLoaded(pos pk.Position) bool {
	w.ChunkLock.RLock()
	defer w.ChunkLock.RUnlock()
	if _, ok := w.Columns[level.ChunkPos{int32(pos.X >> 4), int32(pos.Z >> 4)}]; !ok {
		return false
	}
	return true
}

func (w *World) onPlayerSpawn(pk.Packet) error {
	// unload all chunks
	w.Columns = make(map[level.ChunkPos]*level.Chunk)
	return nil
}

func (w *World) handleLevelChunkWithLightPacket(packet pk.Packet) error {
	var pos level.ChunkPos
	_, currentDimType := w.p.WorldInfo.RegistryCodec.DimensionType.Find(w.p.DimensionName)
	if currentDimType == nil {
		return errors.New("dimension type " + w.p.DimensionName + " not found")
	}
	chunk := level.EmptyChunk(int(currentDimType.Height) / 16)
	if err := packet.Scan(&pos, chunk); err != nil {
		return err
	}
	w.ChunkLock.Lock()
	defer w.ChunkLock.Unlock()
	w.Columns[pos] = chunk
	if w.events.LoadChunk != nil {
		if err := w.events.LoadChunk(pos); err != nil {
			return err
		}
	}
	return nil
}

func (w *World) handleForgetLevelChunkPacket(packet pk.Packet) error {
	var pos level.ChunkPos
	if err := packet.Scan(&pos); err != nil {
		return err
	}
	var err error
	if w.events.UnloadChunk != nil {
		err = w.events.UnloadChunk(pos)
	}
	w.ChunkLock.Lock()
	defer w.ChunkLock.Unlock()
	delete(w.Columns, pos)
	return err
}

func (w *World) handleBlockUpdatePacket(packet pk.Packet) error {
	var (
		pos pk.Position
		ID  pk.VarInt
	)
	if err := packet.Scan(&pos, &ID); err != nil {
		return err
	}

	w.unaryBlockUpdate(pos, level.BlocksState(ID))

	if w.events.BlockUpdate != nil {
		if err := w.events.BlockUpdate(pos.X, pos.Y, pos.Z, level.BlocksState(ID)); err != nil {
			return err
		}
	}

	return nil
}

func (w *World) handleMultiBlockUpdatePacket(packet pk.Packet) error {
	var (
		pos        pk.Long
		TrustEdges pk.Boolean
		Blocks     []pk.VarLong
	)
	if err := packet.Scan(
		&pos,
		&TrustEdges,
		pk.Ary[pk.VarLong]{
			Ary: &Blocks,
		}); err != nil {
		return err
	}
	chunkX := int(pos >> 42)
	chunkY := int(pos << 44 >> 44)
	chunkZ := int(pos << 22 >> 42)
	_, currentDimType := w.p.WorldInfo.RegistryCodec.DimensionType.Find(w.p.DimensionName)
	if currentDimType == nil {
		return errors.New("dimension type " + w.p.DimensionName + " not found")
	}
	chunkY = chunkY - int(currentDimType.MinY>>4)

	w.multiBlockUpdate(chunkX, chunkY, chunkZ, Blocks)

	return nil
}

func (w *World) unaryBlockUpdate(pos pk.Position, bStateID level.BlocksState) bool {
	w.ChunkLock.Lock()
	defer w.ChunkLock.Unlock()

	c := w.Columns[level.ChunkPos{int32(pos.X >> 4), int32(pos.Z >> 4)}]
	if c == nil {
		return false
	}

	_, currentDimType := w.p.WorldInfo.RegistryCodec.DimensionType.Find(w.p.DimensionName)
	if currentDimType == nil {
		return false
	}
	sIdx, bIdx := (int32(pos.Y)-currentDimType.MinY)>>4, sectionIdx(pos.X&15, pos.Y&15, pos.Z&15)

	c.Sections[sIdx].SetBlock(bIdx, bStateID)

	return true
}

func (w *World) multiBlockUpdate(chunkX, chunkY, chunkZ int, blocks []pk.VarLong) bool {
	w.ChunkLock.Lock()
	defer w.ChunkLock.Unlock()

	c := w.Columns[level.ChunkPos{int32(chunkX), int32(chunkZ)}]
	if c == nil {
		return false // not loaded
	}

	section := c.Sections[chunkY]

	for _, b := range blocks {
		var (
			bStateID = level.BlocksState(b >> 12)
			x, z, y  = (b >> 8) & 0xf, (b >> 4) & 0xf, b & 0xf
			bIdx     = sectionIdx(int(x&15), int(y&15), int(z&15))
		)
		section.SetBlock(bIdx, bStateID)

		if w.events.MultiBlockUpdate != nil {
			if err := w.events.MultiBlockUpdate(int(x), int(z), int(y), bStateID); err != nil {
				return false
			}
		}
	}
	return true
}

func sectionIdx(x, y, z int) int {
	return ((y & 15) << 8) | (z << 4) | x
}
