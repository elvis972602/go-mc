package world

import "github.com/Tnze/go-mc/level"

type EventsListener struct {
	LoadChunk        func(pos level.ChunkPos) error
	UnloadChunk      func(pos level.ChunkPos) error
	BlockUpdate      func(x, y, z int, blockID level.BlocksState) error
	MultiBlockUpdate func(x, y, z int, block level.BlocksState) error
}
